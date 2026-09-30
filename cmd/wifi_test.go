// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestWifiStatusLabel(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "IDLE"},
		{1, "NO_SSID_AVAIL"},
		{3, "CONNECTED"},
		{4, "CONNECT_FAILED"},
		{5, "CONNECTION_LOST"},
		{6, "DISCONNECTED"},
		{255, "NO_SHIELD"},
		{42, "UNKNOWN(42)"},
	}
	for _, tt := range tests {
		if got := wifiStatusLabel(tt.in); got != tt.want {
			t.Errorf("wifiStatusLabel(%d) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty stays empty", "", ""},
		{"non-empty is masked", "hunter2", "********"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskSecret(tt.in); got != tt.want {
				t.Errorf("maskSecret(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func wifiSetCmdFixture() *cobra.Command {
	c := &cobra.Command{Use: "set"}
	c.Flags().String("ssid", "", "")
	c.Flags().String("password", "", "")
	c.Flags().Bool("password-stdin", false, "")
	c.Flags().BoolP("yes", "y", false, "")
	c.Flags().String("hostname", "", "")
	c.Flags().Bool("static-ip", false, "")
	c.Flags().Bool("no-static-ip", false, "")
	c.Flags().String("local-ip", "", "")
	c.Flags().String("gateway-ip", "", "")
	c.Flags().String("subnet-mask", "", "")
	c.Flags().String("dns1", "", "")
	c.Flags().String("dns2", "", "")
	c.MarkFlagsMutuallyExclusive("password", "password-stdin")
	c.MarkFlagsMutuallyExclusive("static-ip", "no-static-ip")
	return c
}

func TestRunWifiSetRejectsConflictingStaticIPFlags(t *testing.T) {
	c := wifiSetCmdFixture()
	if err := c.Flags().Set("static-ip", "true"); err != nil {
		t.Fatalf("set static-ip: %v", err)
	}
	if err := c.Flags().Set("no-static-ip", "true"); err != nil {
		t.Fatalf("set no-static-ip: %v", err)
	}

	if err := c.ValidateFlagGroups(); err == nil {
		t.Fatal("ValidateFlagGroups() = nil error; want error for mutually exclusive --static-ip/--no-static-ip")
	}
}

func TestRunWifiSetRejectsOversizedSSID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)
	withCmdTarget(t, srv.URL)

	c := wifiSetCmdFixture()
	longSSID := strings.Repeat("a", maxSSIDLength+1)
	if err := c.Flags().Set("ssid", longSSID); err != nil {
		t.Fatalf("set ssid: %v", err)
	}

	err := runWifiSet(c, nil)
	if err == nil {
		t.Fatal("runWifiSet() = nil error; want error for a 33-character SSID")
	}
}

func TestRunWifiSetAppliesChangedFieldsOnly(t *testing.T) {
	var posted map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ssid": "old", "static_ip_config": false,
			})
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&posted)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)
	withCmdTarget(t, srv.URL)

	c := wifiSetCmdFixture()
	if err := c.Flags().Set("ssid", "newnet"); err != nil {
		t.Fatalf("set ssid: %v", err)
	}
	if err := c.Flags().Set("static-ip", "true"); err != nil {
		t.Fatalf("set static-ip: %v", err)
	}
	if err := c.Flags().Set("local-ip", "192.168.1.50"); err != nil {
		t.Fatalf("set local-ip: %v", err)
	}

	if err := runWifiSet(c, nil); err != nil {
		t.Fatalf("runWifiSet() = %v", err)
	}

	if posted["ssid"] != "newnet" {
		t.Errorf("posted ssid = %v; want %q", posted["ssid"], "newnet")
	}
	if posted["static_ip_config"] != true {
		t.Errorf("posted static_ip_config = %v; want true", posted["static_ip_config"])
	}
	if posted["local_ip"] != "192.168.1.50" {
		t.Errorf("posted local_ip = %v; want %q", posted["local_ip"], "192.168.1.50")
	}
	if _, ok := posted["hostname"]; ok {
		t.Errorf("posted body set hostname; want it untouched since --hostname was not given")
	}
}

func TestRunWifiScanSortsByRSSIDescending(t *testing.T) {
	scanned := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/scanNetworks":
			scanned = true
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case "/rest/listNetworks":
			_ = json.NewEncoder(w).Encode(wifiNetworksResp{Networks: []wifiNetwork{
				{SSID: "weak", RSSI: -80},
				{SSID: "strong", RSSI: -40},
				{SSID: "mid", RSSI: -60},
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	withCmdTarget(t, srv.URL)

	if err := runWifiScan(wifiScanCmd, nil); err != nil {
		t.Fatalf("runWifiScan() = %v", err)
	}
	if !scanned {
		t.Error("runWifiScan() never triggered scanNetworks")
	}
}

func TestRunWifiStatusAndSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/wifiStatus":
			_ = json.NewEncoder(w).Encode(wifiStatusResp{Status: 3, SSID: "home"})
		case "/rest/wifiSettings":
			_ = json.NewEncoder(w).Encode(wifiSettingsResp{SSID: "home", Password: "secret"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	withCmdTarget(t, srv.URL)

	if err := runWifiStatus(wifiStatusCmd, nil); err != nil {
		t.Fatalf("runWifiStatus() = %v", err)
	}
	if err := runWifiSettings(wifiSettingsCmd, nil); err != nil {
		t.Fatalf("runWifiSettings() = %v", err)
	}
}

func TestWarnPlaintextPassword(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		yes      bool
		wantWarn bool
	}{
		{"http warns", "http://tickerbox.local", false, true},
		{"https does not warn", "https://tickerbox.local", false, false},
		{"http with yes skips the warning", "http://tickerbox.local", true, false},
		{"unparsable host does not warn", "://bad", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			warnPlaintextPassword(&buf, tt.host, tt.yes)
			gotWarn := buf.Len() > 0
			if gotWarn != tt.wantWarn {
				t.Errorf("warnPlaintextPassword(%q, yes=%v) wrote %q; want warning=%v", tt.host, tt.yes, buf.String(), tt.wantWarn)
			}
			if gotWarn && !strings.Contains(buf.String(), tt.host) {
				t.Errorf("warning %q does not name the host %q", buf.String(), tt.host)
			}
		})
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("open pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = orig
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}
	return string(out)
}

func wifiSetPasswordServer(t *testing.T, posted *map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
			return
		}
		_ = json.NewDecoder(r.Body).Decode(posted)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRunWifiSetWarnsOnPlaintextHTTP(t *testing.T) {
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
			var posted map[string]any
			srv := wifiSetPasswordServer(t, &posted)
			withCmdTarget(t, srv.URL)

			c := wifiSetCmdFixture()
			if err := c.Flags().Set("password", "hunter22"); err != nil {
				t.Fatalf("set password: %v", err)
			}
			if tt.yes {
				if err := c.Flags().Set("yes", "true"); err != nil {
					t.Fatalf("set yes: %v", err)
				}
			}

			var runErr error
			stderr := captureStderr(t, func() {
				runErr = runWifiSet(c, nil)
			})
			if runErr != nil {
				t.Fatalf("runWifiSet() = %v", runErr)
			}

			gotWarn := stderr != ""
			if gotWarn != tt.wantWarn {
				t.Errorf("runWifiSet() stderr = %q; want warning=%v", stderr, tt.wantWarn)
			}
		})
	}
}

func TestRunWifiSetPasswordStdin(t *testing.T) {
	var posted map[string]any
	srv := wifiSetPasswordServer(t, &posted)
	withCmdTarget(t, srv.URL)

	origStdin := secretStdin
	t.Cleanup(func() { secretStdin = origStdin })
	secretStdin = strings.NewReader("from-stdin\n")

	c := wifiSetCmdFixture()
	if err := c.Flags().Set("password-stdin", "true"); err != nil {
		t.Fatalf("set password-stdin: %v", err)
	}
	if err := c.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("set yes: %v", err)
	}

	if err := runWifiSet(c, nil); err != nil {
		t.Fatalf("runWifiSet() = %v", err)
	}
	if posted["password"] != "from-stdin" {
		t.Errorf("posted password = %v; want %q", posted["password"], "from-stdin")
	}
}

func TestRunWifiSetPasswordAndStdinMutuallyExclusive(t *testing.T) {
	c := wifiSetCmdFixture()
	if err := c.Flags().Set("password", "hunter22"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := c.Flags().Set("password-stdin", "true"); err != nil {
		t.Fatalf("set password-stdin: %v", err)
	}

	if err := c.ValidateFlagGroups(); err == nil {
		t.Fatal("ValidateFlagGroups() = nil error; want error for --password and --password-stdin together")
	}
}
