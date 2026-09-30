// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/openbunny/tickerbox-cli/internal/diff"
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
	captured, err := section.Capture(context.Background(), newClient(), include, withSecrets)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if err := profile.SaveDescribed("office", captured, "", ""); err != nil {
		t.Fatalf("SaveDescribed: %v", err)
	}

	prof, err := profile.Load("office")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	bodies["settingsState"] = `{"brightness":150}`

	deviceNow, err := section.Capture(context.Background(), newClient(), presentSections(prof), true)
	if err != nil {
		t.Fatalf("Capture after drift: %v", err)
	}

	diffs := diff.Diff(deviceNow, prof, presentSections(prof), false)
	want := []diff.FieldDiff{{Section: section.SectionDisplay, Field: "brightness", Device: float64(150), Saved: float64(200)}}
	if !reflect.DeepEqual(diffs, want) {
		t.Errorf("diffs = %+v; want %+v", diffs, want)
	}

	noDrift := diff.Diff(captured, prof, presentSections(prof), false)
	if len(noDrift) != 0 {
		t.Errorf("diff against the just-saved state = %+v; want none", noDrift)
	}
}
