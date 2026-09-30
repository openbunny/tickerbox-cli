// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
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

func TestReadOnlyRestCmds(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		payload any
		cmd     *cobra.Command
	}{
		{"ap status", "apStatus", apStatusPayload{Status: 0, IPAddress: "10.0.0.1", MACAddress: "aa:bb", StationNum: 3}, apStatusCmd},
		{"ap settings", "apSettings", apSettingsPayload{ProvisionMode: 1, SSID: "box"}, apSettingsCmd},
		{"ntp status", "ntpStatus", ntpStatusPayload{Status: 1, Server: "pool.ntp.org"}, ntpStatusCmd},
		{"ntp settings", "ntpSettings", ntpSettingsPayload{Enabled: true, Server: "pool.ntp.org"}, ntpSettingsCmd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := apJSONServer(t, tt.path, tt.payload)
			withCmdTarget(t, srv.URL)

			if err := tt.cmd.RunE(tt.cmd, nil); err != nil {
				t.Fatalf("%s.RunE() = %v", tt.cmd.Name(), err)
			}
		})
	}
}

func TestApStatusCmdPropagatesTransportError(t *testing.T) {
	withCmdTarget(t, "http://127.0.0.1:1")
	timeoutFlag = 50 * time.Millisecond

	if err := apStatusCmd.RunE(apStatusCmd, nil); err == nil {
		t.Fatal("apStatusCmd.RunE() = nil error; want error when the device is unreachable")
	}
}

func apSetServer(t *testing.T, posted *map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/apSettings" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(apSettingsPayload{})
			return
		}
		_ = json.NewDecoder(r.Body).Decode(posted)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func resetApSetFlags(t *testing.T) {
	t.Helper()
	fs := apSetCmd.Flags()
	for _, name := range []string{"password", "password-stdin", "yes", "hidden", "no-hidden", "local-ip", "gateway-ip", "subnet-mask"} {
		f := fs.Lookup(name)
		if err := fs.Set(name, f.DefValue); err != nil {
			t.Fatalf("reset %s: %v", name, err)
		}
		f.Changed = false
	}
}

func TestApSetCmdPasswordValueWarnsOverHTTP(t *testing.T) {
	tests := []struct {
		name     string
		yes      bool
		wantWarn bool
	}{
		{"warns by default", false, true},
		{"yes skips the warning", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetApSetFlags(t)
			var posted map[string]any
			srv := apSetServer(t, &posted)
			withCmdTarget(t, srv.URL)

			if err := apSetCmd.Flags().Set("password", "hunterhunter2"); err != nil {
				t.Fatalf("set password: %v", err)
			}
			if tt.yes {
				if err := apSetCmd.Flags().Set("yes", "true"); err != nil {
					t.Fatalf("set yes: %v", err)
				}
			}

			var runErr error
			stderr := captureStderr(t, func() {
				runErr = apSetCmd.RunE(apSetCmd, nil)
			})
			if runErr != nil {
				t.Fatalf("apSetCmd.RunE() = %v", runErr)
			}
			gotWarn := stderr != ""
			if gotWarn != tt.wantWarn {
				t.Errorf("apSetCmd.RunE() stderr = %q; want warning=%v", stderr, tt.wantWarn)
			}
			if posted["password"] != "hunterhunter2" {
				t.Errorf("posted password = %v; want %q", posted["password"], "hunterhunter2")
			}
		})
	}
}

func TestApSetCmdPasswordStdin(t *testing.T) {
	resetApSetFlags(t)
	var posted map[string]any
	srv := apSetServer(t, &posted)
	withCmdTarget(t, srv.URL)

	origStdin := secretStdin
	t.Cleanup(func() { secretStdin = origStdin })
	secretStdin = strings.NewReader("apsecret1\n")

	if err := apSetCmd.Flags().Set("password-stdin", "true"); err != nil {
		t.Fatalf("set password-stdin: %v", err)
	}
	if err := apSetCmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("set yes: %v", err)
	}

	if err := apSetCmd.RunE(apSetCmd, nil); err != nil {
		t.Fatalf("apSetCmd.RunE() = %v", err)
	}
	if posted["password"] != "apsecret1" {
		t.Errorf("posted password = %v; want %q", posted["password"], "apsecret1")
	}
}

func TestApSetCmdPasswordAndStdinMutuallyExclusive(t *testing.T) {
	resetApSetFlags(t)

	if err := apSetCmd.Flags().Set("password", "hunterhunter2"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := apSetCmd.Flags().Set("password-stdin", "true"); err != nil {
		t.Fatalf("set password-stdin: %v", err)
	}

	if err := apSetCmd.ValidateFlagGroups(); err == nil {
		t.Fatal("ValidateFlagGroups() = nil error; want error for --password and --password-stdin together")
	}
}

func TestApSetCmdHiddenAndNoHiddenMutuallyExclusive(t *testing.T) {
	resetApSetFlags(t)

	if err := apSetCmd.Flags().Set("hidden", "true"); err != nil {
		t.Fatalf("set hidden: %v", err)
	}
	if err := apSetCmd.Flags().Set("no-hidden", "true"); err != nil {
		t.Fatalf("set no-hidden: %v", err)
	}

	if err := apSetCmd.ValidateFlagGroups(); err == nil {
		t.Fatal("ValidateFlagGroups() = nil error; want error for --hidden and --no-hidden together")
	}
}

func resetNtpSetFlags(t *testing.T) {
	t.Helper()
	fs := ntpSetCmd.Flags()
	for _, name := range []string{"enabled", "disabled"} {
		if err := fs.Set(name, fs.Lookup(name).DefValue); err != nil {
			t.Fatalf("reset %s: %v", name, err)
		}
	}
}

func TestNtpSetCmdEnabledAndDisabledMutuallyExclusive(t *testing.T) {
	resetNtpSetFlags(t)
	t.Cleanup(func() { resetNtpSetFlags(t) })

	if err := ntpSetCmd.Flags().Set("enabled", "true"); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	if err := ntpSetCmd.Flags().Set("disabled", "true"); err != nil {
		t.Fatalf("set disabled: %v", err)
	}

	if err := ntpSetCmd.ValidateFlagGroups(); err == nil {
		t.Fatal("ValidateFlagGroups() = nil error; want error for --enabled and --disabled together")
	}
}

func TestApSettingsCmdMasksPasswordByDefault(t *testing.T) {
	srv := apJSONServer(t, "apSettings", apSettingsPayload{SSID: "box-ap", Password: "hunter2"})

	tests := []struct {
		name        string
		showSecrets bool
		want        string
	}{
		{"masked by default", false, "********"},
		{"revealed with --show-secrets", true, "hunter2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withCmdTarget(t, srv.URL)
			jsonFlag = false
			if err := apSettingsCmd.Flags().Set("show-secrets", strconv.FormatBool(tt.showSecrets)); err != nil {
				t.Fatalf("set show-secrets: %v", err)
			}
			t.Cleanup(func() {
				_ = apSettingsCmd.Flags().Set("show-secrets", "false")
			})

			stdout := captureStdout(t, func() {
				if err := apSettingsCmd.RunE(apSettingsCmd, nil); err != nil {
					t.Fatalf("apSettingsCmd.RunE() = %v", err)
				}
			})
			if !strings.Contains(stdout, tt.want) {
				t.Errorf("apSettingsCmd.RunE() stdout = %q; want it to contain %q", stdout, tt.want)
			}
		})
	}
}

func TestApSetCmdPasswordRejectsInvalidLength(t *testing.T) {
	resetApSetFlags(t)
	var posted map[string]any
	srv := apSetServer(t, &posted)
	withCmdTarget(t, srv.URL)

	if err := apSetCmd.Flags().Set("password", "short"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := apSetCmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("set yes: %v", err)
	}

	if err := apSetCmd.RunE(apSetCmd, nil); err == nil {
		t.Fatal("apSetCmd.RunE() = nil error; want error for a too-short --password")
	}
}

func TestApSetCmdRejectsInvalidStaticIPFields(t *testing.T) {
	tests := []struct {
		name  string
		flag  string
		value string
	}{
		{name: "malformed local-ip", flag: "local-ip", value: "not-an-ip"},
		{name: "malformed gateway-ip", flag: "gateway-ip", value: "192.168.1.1x"},
		{name: "malformed subnet-mask", flag: "subnet-mask", value: "255.0.255.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetApSetFlags(t)
			var posted bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(apSettingsPayload{})
					return
				}
				posted = true
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{})
			}))
			t.Cleanup(srv.Close)
			withCmdTarget(t, srv.URL)

			if err := apSetCmd.Flags().Set(tt.flag, tt.value); err != nil {
				t.Fatalf("set %s: %v", tt.flag, err)
			}

			if err := apSetCmd.RunE(apSetCmd, nil); err == nil {
				t.Fatalf("apSetCmd.RunE() with --%s %q = nil error; want error", tt.flag, tt.value)
			}
			if posted {
				t.Errorf("device was posted despite an invalid --%s", tt.flag)
			}
		})
	}
}

func TestApSetCmdAcceptsValidStaticIPFields(t *testing.T) {
	resetApSetFlags(t)
	var posted map[string]any
	srv := apSetServer(t, &posted)
	withCmdTarget(t, srv.URL)

	for flag, value := range map[string]string{
		"local-ip": "192.168.4.1", "gateway-ip": "192.168.4.1", "subnet-mask": "255.255.255.0",
	} {
		if err := apSetCmd.Flags().Set(flag, value); err != nil {
			t.Fatalf("set %s: %v", flag, err)
		}
	}

	if err := apSetCmd.RunE(apSetCmd, nil); err != nil {
		t.Fatalf("apSetCmd.RunE() = %v", err)
	}
	if posted["subnet_mask"] != "255.255.255.0" {
		t.Errorf("posted subnet_mask = %v; want %q", posted["subnet_mask"], "255.255.255.0")
	}
}
