// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/config"
	"github.com/openbunny/tickerbox-cli/internal/section"
)

func TestBackupFileName(t *testing.T) {
	got := backupFileName(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC))
	want := "tickerbox-backup-20260102-150405.json"
	if got != want {
		t.Errorf("backupFileName() = %q; want %q", got, want)
	}
}

func TestBackupFileNameConvertsToUTC(t *testing.T) {
	pst := time.FixedZone("PST", -8*60*60)
	got := backupFileName(time.Date(2026, 1, 2, 7, 4, 5, 0, pst))
	want := "tickerbox-backup-20260102-150405.json"
	if got != want {
		t.Errorf("backupFileName() = %q; want %q (input local time should convert to UTC)", got, want)
	}
}

func newStatusDeviceServer(t *testing.T, bodies map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path[1:]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
}

func TestWriteBackup(t *testing.T) {
	bodies := map[string]string{
		"coinSetupState":  `{"size":0,"types":"","tickers":"","times":"","currency":""}`,
		"settingsState":   `{"brightness":200}`,
		"clockSetupState": `{"enabled":true}`,
		"ntpSettings":     `{"server":"pool.ntp.org"}`,
		"wifiSettings":    `{"ssid":"home","password":"hunter2"}`,
		"apSettings":      `{"ssid":"box-ap","secretKey":"topsecret"}`,
	}
	srv := newStatusDeviceServer(t, bodies)
	defer srv.Close()

	origClock := systemClock
	systemClock = func() time.Time { return time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC) }
	defer func() { systemClock = origClock }()

	t.Chdir(t.TempDir())

	c := client.New(srv.URL+"/", 0)
	name, err := writeBackup(context.Background(), c)
	if err != nil {
		t.Fatalf("writeBackup: %v", err)
	}
	if name != "tickerbox-backup-20260304-050607.json" {
		t.Errorf("writeBackup() name = %q; want tickerbox-backup-20260304-050607.json", name)
	}

	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	var snap section.Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("decode backup file: %v", err)
	}
	if _, ok := snap.Wifi["password"]; ok {
		t.Error("backup exposes the wifi password; writeBackup must always exclude secrets")
	}
	if snap.Wifi["ssid"] != "home" {
		t.Errorf("backup wifi ssid = %v; want home", snap.Wifi["ssid"])
	}

	info, err := os.Stat(name)
	if err != nil {
		t.Fatalf("stat backup file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != backupFilePerm {
		t.Errorf("backup file perm = %o; want %o", perm, backupFilePerm)
	}
}

func TestWriteBackupCaptureError(t *testing.T) {
	srv := newStatusDeviceServer(t, nil)
	defer srv.Close()

	t.Chdir(t.TempDir())

	c := client.New(srv.URL+"/", 0)
	if _, err := writeBackup(context.Background(), c); err == nil {
		t.Fatal("writeBackup() with a failing device: want error, got nil")
	}
}

func TestFetchStatusReport(t *testing.T) {
	bodies := map[string]string{
		"features":        `{"project":true,"ntp":true}`,
		"systemStatus":    `{"esp_platform":"esp32"}`,
		"wifiStatus":      `{"status":3,"ssid":"home"}`,
		"clockSetupState": `{"enabled":true}`,
	}
	srv := newStatusDeviceServer(t, bodies)
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	report := fetchStatusReport(context.Background(), c)

	if report.Features == nil || !report.Features.Project {
		t.Errorf("report.Features = %+v; want Project=true", report.Features)
	}
	if report.WifiStatus == nil || report.WifiStatus.SSID != "home" {
		t.Errorf("report.WifiStatus = %+v; want SSID=home", report.WifiStatus)
	}

	for _, key := range []string{"apStatus", "ntpStatus", "settingsState"} {
		if _, ok := report.Errors[key]; !ok {
			t.Errorf("report.Errors missing %q for an endpoint the test server 404s", key)
		}
	}
	if report.ApStatus != nil {
		t.Error("report.ApStatus should be nil when the endpoint 404s")
	}
}

func TestFetchStatusReportAllAvailableHasNoErrors(t *testing.T) {
	bodies := map[string]string{
		"features":        `{}`,
		"systemStatus":    `{}`,
		"wifiStatus":      `{}`,
		"apStatus":        `{}`,
		"ntpStatus":       `{}`,
		"settingsState":   `{}`,
		"clockSetupState": `{}`,
	}
	srv := newStatusDeviceServer(t, bodies)
	defer srv.Close()

	report := fetchStatusReport(context.Background(), client.New(srv.URL+"/", 0))
	if report.Errors != nil {
		t.Errorf("report.Errors = %v; want nil when every endpoint responds", report.Errors)
	}
}

func TestPrintStatusReportShowsErrorDetail(t *testing.T) {
	report := statusReport{Errors: map[string]string{
		"features":        "get features: timeout",
		"systemStatus":    "get systemStatus: 404",
		"wifiStatus":      "get wifiStatus: 404",
		"apStatus":        "get apStatus: 404",
		"ntpStatus":       "get ntpStatus: 404",
		"settingsState":   "get settingsState: 404",
		"clockSetupState": "get clockSetupState: 404",
	}}

	stdout := captureStdout(t, func() {
		if err := printStatusReport(report); err != nil {
			t.Fatalf("printStatusReport: %v", err)
		}
	})

	for key, detail := range report.Errors {
		if !strings.Contains(stdout, detail) {
			t.Errorf("stdout missing %q's error detail %q; got %q", key, detail, stdout)
		}
	}
	if strings.Count(stdout, "  unavailable\n") != 0 {
		t.Errorf("stdout contains a bare \"unavailable\" with no error detail: %q", stdout)
	}
}

func TestCollectDeviceStatuses(t *testing.T) {
	srvA := newStatusDeviceServer(t, map[string]string{"rest/features": `{"project":true}`})
	defer srvA.Close()
	srvB := newStatusDeviceServer(t, map[string]string{"rest/features": `{"project":false}`})
	defer srvB.Close()

	origTimeout, origRetry := timeoutFlag, retryFlag
	timeoutFlag, retryFlag = time.Second, 0
	defer func() { timeoutFlag, retryFlag = origTimeout, origRetry }()

	devices := []config.Device{
		{Name: "kitchen", Host: srvA.URL},
		{Name: "office", Host: srvB.URL},
	}
	got := collectDeviceStatuses(context.Background(), devices)

	if len(got) != 2 {
		t.Fatalf("collectDeviceStatuses() returned %d reports; want 2", len(got))
	}
	if got[0].Device != "kitchen" || got[0].Report.Features == nil || !got[0].Report.Features.Project {
		t.Errorf("got[0] = %+v; want kitchen with Project=true", got[0])
	}
	if got[1].Device != "office" || got[1].Report.Features == nil || got[1].Report.Features.Project {
		t.Errorf("got[1] = %+v; want office with Project=false", got[1])
	}
}

func useSystemTempConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir)
}

func TestRunStatusAllNoDevicesConfigured(t *testing.T) {
	useSystemTempConfigDir(t)

	if err := runStatusAll(context.Background()); err == nil {
		t.Fatal("runStatusAll() with no configured devices: want error, got nil")
	}
}

func TestRunStatusAllJSON(t *testing.T) {
	useSystemTempConfigDir(t)

	srv := newStatusDeviceServer(t, map[string]string{"rest/features": `{"project":true}`})
	defer srv.Close()

	cfg := &config.Config{Devices: map[string]config.Device{}}
	if err := cfg.Add("kitchen", srv.URL); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	origJSON := jsonFlag
	jsonFlag = true
	defer func() { jsonFlag = origJSON }()

	if err := runStatusAll(context.Background()); err != nil {
		t.Fatalf("runStatusAll: %v", err)
	}
}
