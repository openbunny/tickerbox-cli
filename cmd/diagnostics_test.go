// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
)

func TestEvaluateFreeHeap(t *testing.T) {
	tests := []struct {
		name     string
		freeHeap int64
		want     severity
	}{
		{"just below critical floor", doctorFreeHeapCriticalBytes - 1, severityCritical},
		{"at critical floor", doctorFreeHeapCriticalBytes, severityWarn},
		{"just below warn floor", doctorFreeHeapWarnBytes - 1, severityWarn},
		{"at warn floor", doctorFreeHeapWarnBytes, severityOK},
		{"well above warn floor", doctorFreeHeapWarnBytes * 10, severityOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluateFreeHeap(tt.freeHeap); got.Severity != tt.want.String() {
				t.Errorf("evaluateFreeHeap(%d) severity = %q; want %q", tt.freeHeap, got.Severity, tt.want.String())
			}
		})
	}
}

func TestEvaluateFsHeadroom(t *testing.T) {
	tests := []struct {
		name        string
		total, used int64
		want        severity
	}{
		{"just below critical floor", doctorFsHeadroomCriticalBytes - 1, 0, severityCritical},
		{"at critical floor", doctorFsHeadroomCriticalBytes, 0, severityWarn},
		{"just below warn floor", doctorFsHeadroomWarnBytes - 1, 0, severityWarn},
		{"at warn floor", doctorFsHeadroomWarnBytes, 0, severityOK},
		{"used eats into headroom", doctorFsHeadroomWarnBytes, doctorFsHeadroomWarnBytes - doctorFsHeadroomCriticalBytes + 1, severityCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluateFsHeadroom(tt.total, tt.used); got.Severity != tt.want.String() {
				t.Errorf("evaluateFsHeadroom(%d, %d) severity = %q; want %q", tt.total, tt.used, got.Severity, tt.want.String())
			}
		})
	}
}

func TestEvaluateWifi(t *testing.T) {
	tests := []struct {
		status int
		want   severity
	}{
		{doctorWifiConnectedStatus, severityOK},
		{6, severityCritical},
		{255, severityCritical},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status=%d", tt.status), func(t *testing.T) {
			if got := evaluateWifi(tt.status); got.Severity != tt.want.String() {
				t.Errorf("evaluateWifi(%d) severity = %q; want %q", tt.status, got.Severity, tt.want.String())
			}
		})
	}
}

func TestEvaluateNtp(t *testing.T) {
	tests := []struct {
		status int
		want   severity
	}{
		{doctorNtpActiveStatus, severityOK},
		{0, severityWarn},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status=%d", tt.status), func(t *testing.T) {
			if got := evaluateNtp(tt.status); got.Severity != tt.want.String() {
				t.Errorf("evaluateNtp(%d) severity = %q; want %q", tt.status, got.Severity, tt.want.String())
			}
		})
	}
}

func TestEvaluateApExposure(t *testing.T) {
	tests := []struct {
		status int
		want   severity
	}{
		{doctorApActiveStatus, severityWarn},
		{doctorApLingeringStatus, severityWarn},
		{1, severityOK},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status=%d", tt.status), func(t *testing.T) {
			if got := evaluateApExposure(tt.status); got.Severity != tt.want.String() {
				t.Errorf("evaluateApExposure(%d) severity = %q; want %q", tt.status, got.Severity, tt.want.String())
			}
		})
	}
}

func TestSecretExposureCheckAlwaysWarns(t *testing.T) {
	got := secretExposureCheck()
	if got.Severity != severityWarn.String() {
		t.Errorf("secretExposureCheck().Severity = %q; want %q", got.Severity, severityWarn.String())
	}
	if !strings.Contains(got.Detail, "wifiSettings") || !strings.Contains(got.Detail, "apSettings") || !strings.Contains(got.Detail, "no authentication") {
		t.Errorf("secretExposureCheck().Detail = %q; missing expected content", got.Detail)
	}
}

func TestAggregateSeverity(t *testing.T) {
	tests := []struct {
		name   string
		checks []doctorCheck
		want   severity
	}{
		{"empty", nil, severityOK},
		{"all ok", []doctorCheck{{Severity: severityOK.String()}, {Severity: severityOK.String()}}, severityOK},
		{"one warn", []doctorCheck{{Severity: severityOK.String()}, {Severity: severityWarn.String()}}, severityWarn},
		{"critical wins over warn", []doctorCheck{{Severity: severityWarn.String()}, {Severity: severityCritical.String()}}, severityCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aggregateSeverity(tt.checks); got != tt.want {
				t.Errorf("aggregateSeverity() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestFetchFailureCheckIsCritical(t *testing.T) {
	got := fetchFailureCheck("wifiStatus", errors.New("boom"))
	if got.Severity != severityCritical.String() {
		t.Errorf("fetchFailureCheck().Severity = %q; want %q", got.Severity, severityCritical.String())
	}
	if got.Detail != "boom" {
		t.Errorf("fetchFailureCheck().Detail = %q; want %q", got.Detail, "boom")
	}
}

func doctorTestServer(t *testing.T, handlers map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, payload := range handlers {
		mux.HandleFunc("/rest/"+path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestRunDoctorHealthyDevice(t *testing.T) {
	srv := doctorTestServer(t, map[string]any{
		"features":     featuresPayload{},
		"systemStatus": systemStatusPayload{FreeHeap: doctorFreeHeapWarnBytes * 4, FsTotal: 100000, FsUsed: 10000},
		"wifiStatus":   dashboardWifiStatus{Status: doctorWifiConnectedStatus},
		"apStatus":     dashboardApStatus{Status: 1},
		"ntpStatus":    dashboardNtpStatus{Status: doctorNtpActiveStatus},
	})

	c := client.New(srv.URL+"/rest/", time.Second)
	report := runDoctor(context.Background(), c)

	if report.Verdict != severityWarn.String() {
		t.Fatalf("Verdict = %q; want %q (secret-exposure warning always present)", report.Verdict, severityWarn.String())
	}
	for _, chk := range report.Checks {
		if chk.Severity == severityCritical.String() {
			t.Errorf("unexpected critical check on a healthy device: %+v", chk)
		}
	}
}

func TestRunDoctorLowHeapIsCritical(t *testing.T) {
	srv := doctorTestServer(t, map[string]any{
		"features":     featuresPayload{},
		"systemStatus": systemStatusPayload{FreeHeap: 100, FsTotal: 100000, FsUsed: 10000},
		"wifiStatus":   dashboardWifiStatus{Status: doctorWifiConnectedStatus},
		"apStatus":     dashboardApStatus{Status: 1},
		"ntpStatus":    dashboardNtpStatus{Status: doctorNtpActiveStatus},
	})

	c := client.New(srv.URL+"/rest/", time.Second)
	report := runDoctor(context.Background(), c)

	if report.Verdict != severityCritical.String() {
		t.Fatalf("Verdict = %q; want %q", report.Verdict, severityCritical.String())
	}
}

func TestRunDoctorUnreachableDeviceIsCritical(t *testing.T) {
	c := client.New("http://127.0.0.1:1/rest/", 50*time.Millisecond)
	report := runDoctor(context.Background(), c)

	if report.Verdict != severityCritical.String() {
		t.Fatalf("Verdict = %q; want %q", report.Verdict, severityCritical.String())
	}
}

func TestDoctorCmdExitCode(t *testing.T) {
	tests := []struct {
		name     string
		handlers map[string]any
		wantErr  bool
	}{
		{
			name: "critical check exits non-zero",
			handlers: map[string]any{
				"features":     featuresPayload{},
				"systemStatus": systemStatusPayload{FreeHeap: 1, FsTotal: 1, FsUsed: 0},
				"wifiStatus":   dashboardWifiStatus{Status: 6},
				"apStatus":     dashboardApStatus{Status: doctorApActiveStatus},
				"ntpStatus":    dashboardNtpStatus{Status: 0},
			},
			wantErr: true,
		},
		{
			name: "healthy device succeeds",
			handlers: map[string]any{
				"features":     featuresPayload{},
				"systemStatus": systemStatusPayload{FreeHeap: doctorFreeHeapWarnBytes * 4, FsTotal: 100000, FsUsed: 10000},
				"wifiStatus":   dashboardWifiStatus{Status: doctorWifiConnectedStatus},
				"apStatus":     dashboardApStatus{Status: 1},
				"ntpStatus":    dashboardNtpStatus{Status: doctorNtpActiveStatus},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := doctorTestServer(t, tt.handlers)

			origHost, origTimeout, origRetry, origAll, origJSON := resolvedHost, timeoutFlag, retryFlag, doctorAll, jsonFlag
			t.Cleanup(func() {
				resolvedHost, timeoutFlag, retryFlag, doctorAll, jsonFlag = origHost, origTimeout, origRetry, origAll, origJSON
			})
			resolvedHost, timeoutFlag, retryFlag, doctorAll, jsonFlag = srv.URL, time.Second, 0, false, true

			err := doctorCmd.RunE(doctorCmd, nil)
			if tt.wantErr && err == nil {
				t.Fatal("doctorCmd.RunE() = nil error; want non-nil on a critical check")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("doctorCmd.RunE() = %v; want nil on a healthy device", err)
			}
		})
	}
}

func TestResolveWatchTarget(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *cobra.Command
		wantErr bool
	}{
		{name: "no args defaults to status dashboard", args: nil, want: statusCmd},
		{name: "resolves a nested read-only view", args: []string{"wifi", "status"}, want: wifiStatusCmd},
		{name: "rejects a side-effecting command", args: []string{"system", "restart"}, wantErr: true},
		{name: "rejects an unknown command", args: []string{"bogus"}, wantErr: true},
		{name: "rejects trailing extra args", args: []string{"wifi", "settings", "extra"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveWatchTarget(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveWatchTarget(%v) = nil error; want error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveWatchTarget(%v) unexpected error: %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("resolveWatchTarget(%v) = %s; want %s", tt.args, got.Name(), tt.want.Name())
			}
		})
	}
}

func TestAssembleVersion(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
			{Key: "vcs.time", Value: "2024-01-01T00:00:00Z"},
		},
	}

	tests := []struct {
		name    string
		v, c, d string
		info    *debug.BuildInfo
		ok      bool
		want    string
	}{
		{
			name: "ldflags override wins over build info",
			v:    "1.0.0", c: "deadbeef", d: "2023-06-01",
			info: info, ok: true,
			want: "tickerbox 1.0.0 (commit deadbeef, built 2023-06-01)",
		},
		{
			name: "unset values fall back to build info",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info: info, ok: true,
			want: "tickerbox v1.2.3 (commit abc123, built 2024-01-01T00:00:00Z)",
		},
		{
			name: "no build info available keeps sentinels",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info: nil, ok: false,
			want: "tickerbox dev (commit none, built unknown)",
		},
		{
			name: "devel module version is not used as a fallback",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, ok: true,
			want: "tickerbox dev (commit none, built unknown)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := assembleVersion(tt.v, tt.c, tt.d, tt.info, tt.ok); got != tt.want {
				t.Errorf("assembleVersion() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestAssembleVersionMarksDirtyBuild(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	want := "tickerbox dev (commit abc123-dirty, built unknown)"
	if got := assembleVersion(unsetVersion, unsetCommit, unsetDate, info, true); got != want {
		t.Errorf("assembleVersion() = %q; want %q", got, want)
	}
}

func TestBuildSetting(t *testing.T) {
	info := &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}}}

	if v, ok := buildSetting(info, "vcs.revision"); !ok || v != "abc123" {
		t.Errorf("buildSetting(info, %q) = (%q, %v); want (%q, true)", "vcs.revision", v, ok, "abc123")
	}
	if _, ok := buildSetting(info, "vcs.time"); ok {
		t.Error("buildSetting() found a key that was not set")
	}
	if _, ok := buildSetting(nil, "vcs.revision"); ok {
		t.Error("buildSetting(nil, ...) should report not found")
	}
}

func TestDeviceUIURL(t *testing.T) {
	tests := []struct{ host, want string }{
		{"http://tickerbox.local", "http://tickerbox.local/"},
		{"http://tickerbox.local/", "http://tickerbox.local/"},
		{"http://192.168.10.89", "http://192.168.10.89/"},
	}
	for _, tt := range tests {
		if got := deviceUIURL(tt.host); got != tt.want {
			t.Errorf("deviceUIURL(%q) = %q; want %q", tt.host, got, tt.want)
		}
	}
}

func TestBrowserOpenCommand(t *testing.T) {
	tests := []struct {
		goos     string
		wantName string
		wantOK   bool
	}{
		{goosDarwin, "open", true},
		{goosLinux, "xdg-open", true},
		{goosWindows, "cmd", true},
		{"plan9", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			name, _, ok := browserOpenCommand(tt.goos)
			if name != tt.wantName || ok != tt.wantOK {
				t.Errorf("browserOpenCommand(%q) = (%q, _, %v); want (%q, _, %v)", tt.goos, name, ok, tt.wantName, tt.wantOK)
			}
		})
	}
}
