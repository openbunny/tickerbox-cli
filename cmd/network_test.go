package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApStatusLabel(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "ACTIVE"},
		{1, "INACTIVE"},
		{2, "LINGERING"},
		{99, "UNKNOWN(99)"},
	}
	for _, tt := range tests {
		if got := apStatusLabel(tt.in); got != tt.want {
			t.Errorf("apStatusLabel(%d) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestApModeLabel(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "AP always"},
		{1, "AP when wifi disconnected"},
		{2, "AP never"},
		{7, "UNKNOWN(7)"},
	}
	for _, tt := range tests {
		if got := apModeLabel(tt.in); got != tt.want {
			t.Errorf("apModeLabel(%d) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestNtpStatusLabel(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "INACTIVE"},
		{1, "ACTIVE"},
		{5, "UNKNOWN(5)"},
	}
	for _, tt := range tests {
		if got := ntpStatusLabel(tt.in); got != tt.want {
			t.Errorf("ntpStatusLabel(%d) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseTimeValue(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    time.Time
		wantErr bool
	}{
		{
			name: "RFC3339 with offset",
			in:   "2024-01-01T05:00:00+05:00",
			want: time.Date(2024, 1, 1, 5, 0, 0, 0, time.FixedZone("", 5*3600)),
		},
		{
			name: "device layout, no offset",
			in:   "2024-06-15T12:30:00",
			want: time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC),
		},
		{name: "garbage", in: "not-a-time", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeValue(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseTimeValue(%q) = %v; want %v", tt.in, got, tt.want)
			}
		})
	}
}

func timeCaptureServer(t *testing.T) (*httptest.Server, *[]byte) {
	t.Helper()
	var captured []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		captured = body
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{})
	}))
	t.Cleanup(srv.Close)
	return srv, &captured
}

func withCmdTarget(t *testing.T, host string) {
	t.Helper()
	origHost, origTimeout, origRetry, origJSON := resolvedHost, timeoutFlag, retryFlag, jsonFlag
	t.Cleanup(func() {
		resolvedHost, timeoutFlag, retryFlag, jsonFlag = origHost, origTimeout, origRetry, origJSON
	})
	resolvedHost, timeoutFlag, retryFlag, jsonFlag = host, time.Second, 0, true
}

// TestTimeCmdUsesInjectedClockInUTC feeds a fixed, non-UTC-located clock
// into timeCmd and checks the device receives a UTC-normalized instant,
// never the wall-clock time of the machine running the test.
func TestTimeCmdUsesInjectedClockInUTC(t *testing.T) {
	srv, captured := timeCaptureServer(t)
	withCmdTarget(t, srv.URL)

	fixedZone := time.FixedZone("TEST", 9*3600)
	fixed := time.Date(2030, 3, 4, 13, 0, 0, 0, fixedZone)
	origNow := now
	t.Cleanup(func() { now = origNow })
	now = func() time.Time { return fixed }

	if err := timeCmd.RunE(timeCmd, nil); err != nil {
		t.Fatalf("timeCmd.RunE() = %v", err)
	}

	var body map[string]string
	if err := json.Unmarshal(*captured, &body); err != nil {
		t.Fatalf("unmarshal posted body %q: %v", *captured, err)
	}
	want := "2030-03-04T04:00:00"
	if body["local_time"] != want {
		t.Errorf("posted local_time = %q; want %q (fixed clock normalized to UTC)", body["local_time"], want)
	}
}

// TestTimeCmdExplicitValueNormalizedToUTC checks that an explicit VALUE
// argument carrying a UTC offset is normalized to UTC before it is sent,
// regardless of the injected clock.
func TestTimeCmdExplicitValueNormalizedToUTC(t *testing.T) {
	srv, captured := timeCaptureServer(t)
	withCmdTarget(t, srv.URL)

	if err := timeCmd.RunE(timeCmd, []string{"2024-01-01T05:00:00+05:00"}); err != nil {
		t.Fatalf("timeCmd.RunE() = %v", err)
	}

	var body map[string]string
	if err := json.Unmarshal(*captured, &body); err != nil {
		t.Fatalf("unmarshal posted body %q: %v", *captured, err)
	}
	want := "2024-01-01T00:00:00"
	if body["local_time"] != want {
		t.Errorf("posted local_time = %q; want %q", body["local_time"], want)
	}
}

func TestTimeCmdInvalidValueRejectedBeforeRequest(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	t.Cleanup(srv.Close)
	withCmdTarget(t, srv.URL)

	if err := timeCmd.RunE(timeCmd, []string{"not-a-time"}); err == nil {
		t.Fatal("timeCmd.RunE() = nil error; want error for an unparsable VALUE")
	}
	if requests != 0 {
		t.Errorf("device received %d requests; want 0 for a rejected VALUE", requests)
	}
}

func apJSONServer(t *testing.T, path string, payload any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/"+path {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestApStatusCmd(t *testing.T) {
	srv := apJSONServer(t, "apStatus", apStatusPayload{Status: 0, IPAddress: "10.0.0.1", MACAddress: "aa:bb", StationNum: 3})
	withCmdTarget(t, srv.URL)

	if err := apStatusCmd.RunE(apStatusCmd, nil); err != nil {
		t.Fatalf("apStatusCmd.RunE() = %v", err)
	}
}

func TestApSettingsCmd(t *testing.T) {
	srv := apJSONServer(t, "apSettings", apSettingsPayload{ProvisionMode: 1, SSID: "box"})
	withCmdTarget(t, srv.URL)

	if err := apSettingsCmd.RunE(apSettingsCmd, nil); err != nil {
		t.Fatalf("apSettingsCmd.RunE() = %v", err)
	}
}

func TestNtpStatusCmd(t *testing.T) {
	srv := apJSONServer(t, "ntpStatus", ntpStatusPayload{Status: 1, Server: "pool.ntp.org"})
	withCmdTarget(t, srv.URL)

	if err := ntpStatusCmd.RunE(ntpStatusCmd, nil); err != nil {
		t.Fatalf("ntpStatusCmd.RunE() = %v", err)
	}
}

func TestNtpSettingsCmd(t *testing.T) {
	srv := apJSONServer(t, "ntpSettings", ntpSettingsPayload{Enabled: true, Server: "pool.ntp.org"})
	withCmdTarget(t, srv.URL)

	if err := ntpSettingsCmd.RunE(ntpSettingsCmd, nil); err != nil {
		t.Fatalf("ntpSettingsCmd.RunE() = %v", err)
	}
}

func TestApStatusCmdPropagatesTransportError(t *testing.T) {
	withCmdTarget(t, "http://127.0.0.1:1")
	timeoutFlag = 50 * time.Millisecond

	if err := apStatusCmd.RunE(apStatusCmd, nil); err == nil {
		t.Fatal("apStatusCmd.RunE() = nil error; want error when the device is unreachable")
	}
}
