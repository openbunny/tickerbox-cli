package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func TestConfigIsSecretField(t *testing.T) {
	tests := []struct {
		field string
		want  bool
	}{
		{"password", true},
		{"Password", true},
		{"secretKey", true},
		{"SECRET", true},
		{"ssid", false},
		{"channel", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if got := configIsSecretField(tt.field); got != tt.want {
				t.Errorf("configIsSecretField(%q) = %v; want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestConfigDiffMapSection(t *testing.T) {
	tests := []struct {
		name   string
		device map[string]any
		saved  map[string]any
		want   []FieldDiff
	}{
		{
			name:   "identical maps produce no diffs",
			device: map[string]any{"a": 1.0, "b": "x"},
			saved:  map[string]any{"a": 1.0, "b": "x"},
			want:   nil,
		},
		{
			name:   "changed value",
			device: map[string]any{"brightness": 200.0},
			saved:  map[string]any{"brightness": 150.0},
			want:   []FieldDiff{{Section: "display", Field: "brightness", Device: 200.0, Saved: 150.0}},
		},
		{
			name:   "key only on device",
			device: map[string]any{"a": 1.0, "b": 2.0},
			saved:  map[string]any{"a": 1.0},
			want:   []FieldDiff{{Section: "display", Field: "b", Device: 2.0, Saved: nil}},
		},
		{
			name:   "key only on saved",
			device: map[string]any{"a": 1.0},
			saved:  map[string]any{"a": 1.0, "b": 2.0},
			want:   []FieldDiff{{Section: "display", Field: "b", Device: nil, Saved: 2.0}},
		},
		{
			name:   "both nil",
			device: nil,
			saved:  nil,
			want:   nil,
		},
		{
			name:   "results sorted by field name",
			device: map[string]any{"z": 1.0, "a": 1.0},
			saved:  map[string]any{"z": 2.0, "a": 2.0},
			want: []FieldDiff{
				{Section: "display", Field: "a", Device: 1.0, Saved: 2.0},
				{Section: "display", Field: "z", Device: 1.0, Saved: 2.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffMapSection("display", tt.device, tt.saved)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("diffMapSection() = %+v; want %+v", got, tt.want)
			}
		})
	}
}

func TestConfigDiffTickers(t *testing.T) {
	tests := []struct {
		name   string
		device []tickers.Entry
		saved  []tickers.Entry
		want   []FieldDiff
	}{
		{
			name:   "identical lists produce no diffs",
			device: []tickers.Entry{{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "USD"}},
			saved:  []tickers.Entry{{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "USD"}},
			want:   nil,
		},
		{
			name:   "one field changed at index 0",
			device: []tickers.Entry{{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "USD"}},
			saved:  []tickers.Entry{{Type: "crypto", Ticker: "BTC", Time: "1min", Currency: "USD"}},
			want:   []FieldDiff{{Section: section.SectionTickers, Field: "tickers[0].time", Device: "5min", Saved: "1min"}},
		},
		{
			name:   "device has an extra entry",
			device: []tickers.Entry{{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "USD"}},
			saved:  nil,
			want: []FieldDiff{
				{Section: section.SectionTickers, Field: "tickers[0].type", Device: "crypto", Saved: nil},
				{Section: section.SectionTickers, Field: "tickers[0].ticker", Device: "BTC", Saved: nil},
				{Section: section.SectionTickers, Field: "tickers[0].time", Device: "5min", Saved: nil},
				{Section: section.SectionTickers, Field: "tickers[0].currency", Device: "USD", Saved: nil},
			},
		},
		{
			name:   "both empty",
			device: nil,
			saved:  nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configDiffTickers(tt.device, tt.saved)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("configDiffTickers() = %+v; want %+v", got, tt.want)
			}
		})
	}
}

func TestConfigDiffSnapshotsOrdersBySectionThenField(t *testing.T) {
	device := &section.Snapshot{
		Display: map[string]any{"brightness": 200.0},
		Wifi:    map[string]any{"ssid": "home"},
	}
	saved := &section.Snapshot{
		Display: map[string]any{"brightness": 150.0},
		Wifi:    map[string]any{"ssid": "office"},
	}

	got := configDiffSnapshots(device, saved)
	want := []FieldDiff{
		{Section: section.SectionDisplay, Field: "brightness", Device: 200.0, Saved: 150.0},
		{Section: section.SectionWifi, Field: "ssid", Device: "home", Saved: "office"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("configDiffSnapshots() = %+v; want %+v", got, want)
	}
}

func TestConfigFormatDiffValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil is unset", nil, unsetFieldDisplay},
		{"string passes through", "home", "home"},
		{"float formats without decoration", 200.0, "200"},
		{"bool formats", true, "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := configFormatDiffValue(tt.in); got != tt.want {
				t.Errorf("configFormatDiffValue(%v) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRedactSnapshotSecrets(t *testing.T) {
	original := &section.Snapshot{
		Wifi: map[string]any{"ssid": "home", "password": "hunter2"},
		AP:   map[string]any{"ssid": "box-ap", "secretKey": "topsecret"},
		NTP:  map[string]any{"server": "pool.ntp.org"},
	}

	got := redactSnapshotSecrets(original)

	if !reflect.DeepEqual(got.Wifi, map[string]any{"ssid": "home"}) {
		t.Errorf("redacted wifi = %v; want ssid only", got.Wifi)
	}
	if !reflect.DeepEqual(got.AP, map[string]any{"ssid": "box-ap"}) {
		t.Errorf("redacted ap = %v; want ssid only", got.AP)
	}
	if !reflect.DeepEqual(got.NTP, map[string]any{"server": "pool.ntp.org"}) {
		t.Errorf("ntp section changed by redaction: %v", got.NTP)
	}
	if original.Wifi["password"] != "hunter2" {
		t.Error("redactSnapshotSecrets mutated the original snapshot's wifi map")
	}
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

func TestLoadSnapshotInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	if _, err := loadSnapshot(path); err == nil {
		t.Fatal("loadSnapshot() with invalid JSON: want error, got nil")
	}
}

func TestLoadSnapshotMissingFile(t *testing.T) {
	if _, err := loadSnapshot(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("loadSnapshot() with missing file: want error, got nil")
	}
}

// TestExportImportRoundTrip captures a snapshot from an httptest device,
// writes it as config export does, reads it back as config import does, and
// applies it to a second httptest device, asserting the posted payloads
// match what was captured.
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
	captured, err := section.Capture(deviceClient, section.Sections, false)
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
	if err := section.Apply(applyClient, loaded, include); err != nil {
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

	// configExportShowSecrets defaults to false: this mirrors the value
	// section.Capture receives from a bare `config export` invocation.
	snap, err := section.Capture(c, []string{section.SectionWifi, section.SectionAP}, configExportShowSecrets)
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
