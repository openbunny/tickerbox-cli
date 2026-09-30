// SPDX-License-Identifier: MIT

package diff

import (
	"reflect"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func TestDiffMapFieldChanges(t *testing.T) {
	device := &section.Snapshot{Display: map[string]any{"brightness": 200.0}}
	saved := &section.Snapshot{Display: map[string]any{"brightness": 150.0}}

	got := Diff(device, saved, []string{section.SectionDisplay}, false)
	want := []FieldDiff{{Section: section.SectionDisplay, Field: "brightness", Device: 200.0, Saved: 150.0}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff() = %+v; want %+v", got, want)
	}
}

func TestDiffMapIdenticalFieldProducesNoRow(t *testing.T) {
	device := &section.Snapshot{Display: map[string]any{"brightness": 200.0}}
	saved := &section.Snapshot{Display: map[string]any{"brightness": 200.0}}

	got := Diff(device, saved, []string{section.SectionDisplay}, false)
	if len(got) != 0 {
		t.Errorf("Diff() = %+v; want none", got)
	}
}

func TestDiffMapFieldOnOneSideOnly(t *testing.T) {
	device := &section.Snapshot{Display: map[string]any{"brightness": 200.0}}
	saved := &section.Snapshot{Display: map[string]any{}}

	got := Diff(device, saved, []string{section.SectionDisplay}, false)
	want := []FieldDiff{{Section: section.SectionDisplay, Field: "brightness", Device: 200.0, Saved: nil}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff() = %+v; want %+v", got, want)
	}
}

func TestDiffSecretFieldChangedIsMaskedByDefault(t *testing.T) {
	device := &section.Snapshot{Wifi: map[string]any{"password": "old-pass"}}
	saved := &section.Snapshot{Wifi: map[string]any{"password": "new-pass"}}

	got := Diff(device, saved, []string{section.SectionWifi}, false)
	want := []FieldDiff{{Section: section.SectionWifi, Field: "password", Device: "********", Saved: "********"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff() masked = %+v; want %+v", got, want)
	}

	revealed := Diff(device, saved, []string{section.SectionWifi}, true)
	wantRevealed := []FieldDiff{{Section: section.SectionWifi, Field: "password", Device: "old-pass", Saved: "new-pass"}}
	if !reflect.DeepEqual(revealed, wantRevealed) {
		t.Errorf("Diff() with showSecrets = %+v; want %+v", revealed, wantRevealed)
	}
}

// TestDiffSecretFieldIdenticalProducesNoRow pins the fix for the old profile.go
// engine's permanent false positive: comparing real values before masking means an
// unchanged secret never produces a row, instead of one that always looked changed
// because the device side was captured with the secret already stripped out.
func TestDiffSecretFieldIdenticalProducesNoRow(t *testing.T) {
	device := &section.Snapshot{Wifi: map[string]any{"password": "same-pass"}}
	saved := &section.Snapshot{Wifi: map[string]any{"password": "same-pass"}}

	got := Diff(device, saved, []string{section.SectionWifi}, false)
	if len(got) != 0 {
		t.Errorf("Diff() = %+v; want none for an unchanged secret", got)
	}
}

// TestDiffSecretFieldDifferingIsNeverSilentlyDropped pins the fix for the old
// config.go engine, which stripped secret keys from both sides before comparing,
// so a genuinely differing secret produced no row at all.
func TestDiffSecretFieldDifferingIsNeverSilentlyDropped(t *testing.T) {
	device := &section.Snapshot{AP: map[string]any{"secretKey": "device-key"}}
	saved := &section.Snapshot{AP: map[string]any{"secretKey": "saved-key"}}

	got := Diff(device, saved, []string{section.SectionAP}, false)
	if len(got) != 1 {
		t.Fatalf("Diff() = %+v; want exactly one row for a differing secret", got)
	}
	if got[0].Device != "********" || got[0].Saved != "********" {
		t.Errorf("Diff() row = %+v; want masked device/saved values", got[0])
	}
}

func TestDiffTickersLengthMismatch(t *testing.T) {
	stocksAAPL := tickers.Entry{Type: tickers.TypeStocks, Ticker: "AAPL", Time: tickers.Time15Min, Currency: tickers.CurrencyUSD}
	cryptoBTC := tickers.Entry{Type: tickers.TypeCrypto, Ticker: "BTC", Time: tickers.Time1Min, Currency: tickers.CurrencyUSD}

	tests := []struct {
		name   string
		device []tickers.Entry
		saved  []tickers.Entry
		want   int
	}{
		{name: "identical lists produce no diffs", device: []tickers.Entry{stocksAAPL}, saved: []tickers.Entry{stocksAAPL}, want: 0},
		{name: "device longer", device: []tickers.Entry{stocksAAPL, cryptoBTC}, saved: []tickers.Entry{stocksAAPL}, want: 4},
		{name: "saved longer", device: []tickers.Entry{stocksAAPL}, saved: []tickers.Entry{stocksAAPL, cryptoBTC}, want: 4},
		{name: "equal length, one differing entry", device: []tickers.Entry{stocksAAPL}, saved: []tickers.Entry{cryptoBTC}, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &section.Snapshot{Tickers: tt.device}
			saved := &section.Snapshot{Tickers: tt.saved}
			got := Diff(device, saved, []string{section.SectionTickers}, false)
			if len(got) != tt.want {
				t.Errorf("Diff() = %+v; want %d row(s)", got, tt.want)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil renders as a dash", nil, "-"},
		{"a string renders as itself", "home", "home"},
		{"a float renders without decoration", 200.0, "200"},
		{"a bool renders", true, "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatValue(tt.in); got != tt.want {
				t.Errorf("FormatValue(%v) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}
