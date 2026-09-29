package cmd

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/openbunny/tickerbox-cli/internal/profile"
	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func withCmdTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
}

func TestResolveInclude(t *testing.T) {
	tests := []struct {
		name        string
		includeFlag string
		all         bool
		wantInclude []string
		wantSecrets bool
	}{
		{
			name:        "no flags uses the default include set",
			includeFlag: "",
			all:         false,
			wantInclude: section.DefaultInclude,
			wantSecrets: false,
		},
		{
			name:        "explicit include, no all: secrets stay stripped",
			includeFlag: "tickers,display",
			all:         false,
			wantInclude: []string{"tickers", "display"},
			wantSecrets: false,
		},
		{
			name:        "include list trims whitespace and drops empty tokens",
			includeFlag: " tickers ,, display ,",
			all:         false,
			wantInclude: []string{"tickers", "display"},
			wantSecrets: false,
		},
		{
			name:        "all with no include adds wifi and ap to the default set",
			includeFlag: "",
			all:         true,
			wantInclude: []string{section.SectionTickers, section.SectionDisplay, section.SectionClock, section.SectionNTP, section.SectionWifi, section.SectionAP},
			wantSecrets: true,
		},
		{
			name:        "all with an explicit include appends wifi/ap once",
			includeFlag: "display",
			all:         true,
			wantInclude: []string{"display", section.SectionWifi, section.SectionAP},
			wantSecrets: true,
		},
		{
			name:        "all does not duplicate a wifi/ap already named in include",
			includeFlag: "wifi,display",
			all:         true,
			wantInclude: []string{"wifi", "display", section.SectionAP},
			wantSecrets: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			include, secrets := resolveInclude(tt.includeFlag, tt.all)
			if !reflect.DeepEqual(include, tt.wantInclude) {
				t.Errorf("include = %v; want %v", include, tt.wantInclude)
			}
			if secrets != tt.wantSecrets {
				t.Errorf("withSecrets = %v; want %v", secrets, tt.wantSecrets)
			}
		})
	}
}

func TestPresentSections(t *testing.T) {
	tests := []struct {
		name string
		s    *section.Snapshot
		want []string
	}{
		{
			name: "empty snapshot has no present sections",
			s:    &section.Snapshot{},
			want: nil,
		},
		{
			name: "only captured sections are present, in canonical order",
			s: &section.Snapshot{
				NTP:     map[string]any{"server": "pool.ntp.org"},
				Display: map[string]any{"brightness": float64(200)},
				Tickers: []tickers.Entry{{Type: tickers.TypeStocks, Ticker: "AAPL", Time: tickers.Time15Min, Currency: tickers.CurrencyUSD}},
			},
			want: []string{section.SectionTickers, section.SectionDisplay, section.SectionNTP},
		},
		{
			name: "an empty-but-non-nil map still counts as present",
			s:    &section.Snapshot{Wifi: map[string]any{}},
			want: []string{section.SectionWifi},
		},
		{
			name: "a zero-length ticker slice does not count as present",
			s:    &section.Snapshot{Tickers: []tickers.Entry{}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := presentSections(tt.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("presentSections() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestDiffMap(t *testing.T) {
	tests := []struct {
		name        string
		device      map[string]any
		profile     map[string]any
		showSecrets bool
		want        []fieldDiff
	}{
		{
			name:    "identical maps produce no diffs",
			device:  map[string]any{"brightness": float64(200)},
			profile: map[string]any{"brightness": float64(200)},
			want:    nil,
		},
		{
			name:    "a changed field is reported with both values",
			device:  map[string]any{"brightness": float64(150)},
			profile: map[string]any{"brightness": float64(200)},
			want:    []fieldDiff{{Section: section.SectionDisplay, Field: "brightness", Device: float64(150), Profile: float64(200)}},
		},
		{
			name:    "a field missing on one side reports the other as nil",
			device:  map[string]any{"brightness": float64(150)},
			profile: map[string]any{"brightness": float64(150), "changeInterval": float64(30)},
			want:    []fieldDiff{{Section: section.SectionDisplay, Field: "changeInterval", Device: nil, Profile: float64(30)}},
		},
		{
			name:        "a changed password field is masked by default",
			device:      map[string]any{"ssid": "home", "password": "old-pass"},
			profile:     map[string]any{"ssid": "home", "password": "new-pass"},
			showSecrets: false,
			want:        []fieldDiff{{Section: section.SectionWifi, Field: "password", Device: "********", Profile: "********"}},
		},
		{
			name:        "show-secrets reveals the real values",
			device:      map[string]any{"password": "old-pass"},
			profile:     map[string]any{"password": "new-pass"},
			showSecrets: true,
			want:        []fieldDiff{{Section: section.SectionWifi, Field: "password", Device: "old-pass", Profile: "new-pass"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sectionName := section.SectionWifi
			if _, ok := tt.device["brightness"]; ok {
				sectionName = section.SectionDisplay
			}
			if _, ok := tt.profile["changeInterval"]; ok {
				sectionName = section.SectionDisplay
			}
			got := diffMap(sectionName, tt.device, tt.profile, tt.showSecrets)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("diffMap() = %+v; want %+v", got, tt.want)
			}
		})
	}
}

func TestDiffTickers(t *testing.T) {
	stocksAAPL := tickers.Entry{Type: tickers.TypeStocks, Ticker: "AAPL", Time: tickers.Time15Min, Currency: tickers.CurrencyUSD}
	cryptoBTC := tickers.Entry{Type: tickers.TypeCrypto, Ticker: "BTC", Time: tickers.Time1Min, Currency: tickers.CurrencyUSD}

	tests := []struct {
		name    string
		device  []tickers.Entry
		profile []tickers.Entry
		want    []fieldDiff
	}{
		{
			name:    "identical lists produce no diffs",
			device:  []tickers.Entry{stocksAAPL},
			profile: []tickers.Entry{stocksAAPL},
			want:    nil,
		},
		{
			name:    "a changed entry is reported by index",
			device:  []tickers.Entry{stocksAAPL},
			profile: []tickers.Entry{cryptoBTC},
			want:    []fieldDiff{{Section: section.SectionTickers, Field: "[0]", Device: stocksAAPL, Profile: cryptoBTC}},
		},
		{
			name:    "an extra device entry reports a nil profile side",
			device:  []tickers.Entry{stocksAAPL, cryptoBTC},
			profile: []tickers.Entry{stocksAAPL},
			want:    []fieldDiff{{Section: section.SectionTickers, Field: "[1]", Device: cryptoBTC, Profile: nil}},
		},
		{
			name:    "an extra profile entry reports a nil device side",
			device:  []tickers.Entry{stocksAAPL},
			profile: []tickers.Entry{stocksAAPL, cryptoBTC},
			want:    []fieldDiff{{Section: section.SectionTickers, Field: "[1]", Device: nil, Profile: cryptoBTC}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffTickers(tt.device, tt.profile)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("diffTickers() = %+v; want %+v", got, tt.want)
			}
		})
	}
}

func newDeviceServer(t *testing.T, bodies map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/rest/"):]
		body, ok := bodies[path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestSaveThenDiffDetectsDeviceDrift exercises save, load, and diff together
// against a fake device (httptest) and a redirected profile directory (a
// temp $HOME), with no real device involved.
func TestSaveThenDiffDetectsDeviceDrift(t *testing.T) {
	withCmdTempHome(t)

	bodies := map[string]string{
		"coinSetupState":  `{"size":1,"types":"stocks","tickers":"AAPL","times":"15min","currency":"USD"}`,
		"settingsState":   `{"brightness":200}`,
		"clockSetupState": `{"enabled":true}`,
		"ntpSettings":     `{"server":"pool.ntp.org"}`,
	}
	srv := newDeviceServer(t, bodies)

	origHost, origTimeout, origRetry := resolvedHost, timeoutFlag, retryFlag
	t.Cleanup(func() { resolvedHost, timeoutFlag, retryFlag = origHost, origTimeout, origRetry })
	resolvedHost, timeoutFlag, retryFlag = srv.URL, time.Second, 0

	include, withSecrets := resolveInclude("", false)
	captured, err := section.Capture(newClient(), include, withSecrets)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if err := profile.Save("office", captured); err != nil {
		t.Fatalf("Save: %v", err)
	}

	prof, err := profile.Load("office")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	bodies["settingsState"] = `{"brightness":150}`

	deviceNow, err := section.Capture(newClient(), presentSections(prof), false)
	if err != nil {
		t.Fatalf("Capture after drift: %v", err)
	}

	diffs := diffSnapshots(deviceNow, prof, presentSections(prof), false)
	want := []fieldDiff{{Section: section.SectionDisplay, Field: "brightness", Device: float64(150), Profile: float64(200)}}
	if !reflect.DeepEqual(diffs, want) {
		t.Errorf("diffs = %+v; want %+v", diffs, want)
	}

	noDrift := diffSnapshots(captured, prof, presentSections(prof), false)
	if len(noDrift) != 0 {
		t.Errorf("diff against the just-saved state = %+v; want none", noDrift)
	}
}

func TestFormatDiffValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "nil renders as a dash", in: nil, want: "-"},
		{name: "a string renders as itself", in: "home", want: "home"},
		{name: "a float renders without quotes", in: float64(200), want: "200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDiffValue(tt.in); got != tt.want {
				t.Errorf("formatDiffValue(%v) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}
