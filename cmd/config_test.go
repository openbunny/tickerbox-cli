// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/diff"
	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	f()
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(out)
}

func TestConfigPresentSections(t *testing.T) {
	s := &section.Snapshot{
		Display: map[string]any{"brightness": 200.0},
		Tickers: []tickers.Entry{{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "USD"}},
	}
	got := configPresentSections(s)
	want := []string{section.SectionTickers, section.SectionDisplay}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("configPresentSections() = %v; want %v", got, want)
	}
}

func TestConfigPresentSectionsEmptySnapshot(t *testing.T) {
	if got := configPresentSections(&section.Snapshot{}); got != nil {
		t.Errorf("configPresentSections(empty) = %v; want nil", got)
	}
}

func TestLoadSnapshotRejectsBadInput(t *testing.T) {
	tests := []struct {
		name     string
		writeBad bool
	}{
		{"invalid JSON", true},
		{"missing file", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "snapshot.json")
			if tt.writeBad {
				if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
					t.Fatalf("write test file: %v", err)
				}
			}
			if _, err := loadSnapshot(path); err == nil {
				t.Fatalf("loadSnapshot() with %s: want error, got nil", tt.name)
			}
		})
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	bodies := map[string]string{
		"coinSetupState":  `{"size":1,"types":"crypto","tickers":"BTC","times":"5min","currency":"USD"}`,
		"settingsState":   `{"brightness":200}`,
		"clockSetupState": `{"enabled":true}`,
		"ntpSettings":     `{"server":"pool.ntp.org"}`,
		"wifiSettings":    `{"ssid":"home","password":"hunter2"}`,
		"apSettings":      `{"ssid":"box-ap","secretKey":"topsecret"}`,
	}
	deviceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path[1:]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer deviceSrv.Close()

	deviceClient := client.New(deviceSrv.URL+"/", 0)
	captured, err := section.Capture(context.Background(), deviceClient, section.Sections, false)
	if err != nil {
		t.Fatalf("section.Capture: %v", err)
	}
	if captured.Wifi["password"] != nil {
		t.Fatalf("captured wifi carries a secret with withSecrets=false: %v", captured.Wifi)
	}

	encoded, err := json.MarshalIndent(captured, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	path := filepath.Join(t.TempDir(), "snapshot.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	loaded, err := loadSnapshot(path)
	if err != nil {
		t.Fatalf("loadSnapshot: %v", err)
	}
	if !reflect.DeepEqual(loaded, captured) {
		t.Fatalf("loadSnapshot() = %+v; want %+v", loaded, captured)
	}

	include := configPresentSections(loaded)
	wantInclude := []string{section.SectionTickers, section.SectionDisplay, section.SectionClock, section.SectionNTP, section.SectionWifi, section.SectionAP}
	if !reflect.DeepEqual(include, wantInclude) {
		t.Fatalf("configPresentSections() = %v; want %v", include, wantInclude)
	}

	posted := map[string][]byte{}
	applySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		posted[r.URL.Path[1:]] = body
		w.WriteHeader(http.StatusOK)
	}))
	defer applySrv.Close()

	applyClient := client.New(applySrv.URL+"/", 0)
	if err := section.Apply(context.Background(), applyClient, loaded, include); err != nil {
		t.Fatalf("section.Apply: %v", err)
	}

	var gotDisplay map[string]any
	if err := json.Unmarshal(posted["settingsState"], &gotDisplay); err != nil {
		t.Fatalf("decode posted settingsState: %v", err)
	}
	if gotDisplay["brightness"] != 200.0 {
		t.Errorf("posted settingsState brightness = %v; want 200", gotDisplay["brightness"])
	}
}

func TestExportRedactsSecretsByDefault(t *testing.T) {
	bodies := map[string]string{
		"wifiSettings": `{"ssid":"home","password":"hunter2"}`,
		"apSettings":   `{"ssid":"box-ap","secretKey":"topsecret"}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path[1:]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)

	snap, err := section.Capture(context.Background(), c, []string{section.SectionWifi, section.SectionAP}, configExportShowSecrets)
	if err != nil {
		t.Fatalf("section.Capture: %v", err)
	}
	if _, ok := snap.Wifi["password"]; ok {
		t.Error("default export exposes the wifi password")
	}
	if _, ok := snap.AP["secretKey"]; ok {
		t.Error("default export exposes the ap secret key")
	}
}

func TestConfigDiffCmdMasksChangedSecret(t *testing.T) {
	bodies := map[string]string{
		"coinSetupState":  `{"size":0,"types":"","tickers":"","times":"","currency":""}`,
		"settingsState":   `{}`,
		"clockSetupState": `{}`,
		"ntpSettings":     `{}`,
		"wifiSettings":    `{"ssid":"home","password":"device-pass"}`,
		"apSettings":      `{}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path[len("/rest/"):]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	origHost, origTimeout, origRetry, origJSON := resolvedHost, timeoutFlag, retryFlag, jsonFlag
	t.Cleanup(func() { resolvedHost, timeoutFlag, retryFlag, jsonFlag = origHost, origTimeout, origRetry, origJSON })
	resolvedHost, timeoutFlag, retryFlag, jsonFlag = srv.URL, time.Second, 0, true

	path := filepath.Join(t.TempDir(), "saved.json")
	saved := &section.Snapshot{Wifi: map[string]any{"ssid": "home", "password": "saved-pass"}}
	encoded, err := json.Marshal(saved)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("write saved snapshot: %v", err)
	}

	origShowSecrets := configDiffShowSecrets
	t.Cleanup(func() { configDiffShowSecrets = origShowSecrets })
	configDiffShowSecrets = false

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = configDiffCmd.RunE(configDiffCmd, []string{path})
	})
	if runErr != nil {
		t.Fatalf("configDiffCmd.RunE() = %v", runErr)
	}

	var got []diff.FieldDiff
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("unmarshal diff output %q: %v", stdout, err)
	}
	found := false
	for _, d := range got {
		if d.Field != "password" {
			continue
		}
		found = true
		if d.Device != "********" || d.Saved != "********" {
			t.Errorf("password diff = %+v; want masked device/saved values", d)
		}
	}
	if !found {
		t.Errorf("diff output %+v: want a row for the differing, masked password field (not silently dropped)", got)
	}
}

func FuzzLoadSnapshot(f *testing.F) {
	f.Add(`{}`)
	f.Add(`{"wifi":{"ssid":"home","password":"x"}}`)
	f.Add(`{"tickers":[{"type":"crypto","ticker":"BTC","time":"5min","currency":"USD"}]}`)
	f.Add(`not json`)
	f.Add(`{"tickers": "not an array"}`)
	f.Add(``)

	f.Fuzz(func(t *testing.T, content string) {
		path := filepath.Join(t.TempDir(), "fuzz.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write fuzz file: %v", err)
		}
		snap, err := loadSnapshot(path)
		if err != nil {
			if snap != nil {
				t.Fatalf("loadSnapshot returned a snapshot alongside error %v", err)
			}
			return
		}
		configPresentSections(snap)
	})
}
