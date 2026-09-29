// SPDX-License-Identifier: MIT

package cmd

import (
	"reflect"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func entry(ticker, typ, time, currency string) tickers.Entry {
	return tickers.Entry{Type: typ, Ticker: ticker, Time: time, Currency: currency}
}

func TestResolveEntryIndex(t *testing.T) {
	entries := []tickers.Entry{
		entry("BTC", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
		entry("AAPL", tickers.TypeStocks, tickers.Time5Min, tickers.CurrencyUSD),
	}

	tests := []struct {
		name     string
		selector string
		want     int
		wantErr  bool
	}{
		{name: "by index", selector: "1", want: 1},
		{name: "by symbol", selector: "btc", want: 0},
		{name: "index out of range", selector: "5", wantErr: true},
		{name: "unknown symbol", selector: "ETH", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveEntryIndex(entries, tt.selector)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveEntryIndex(%q) = %d, nil; want error", tt.selector, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveEntryIndex(%q): unexpected error: %v", tt.selector, err)
			}
			if got != tt.want {
				t.Errorf("resolveEntryIndex(%q) = %d; want %d", tt.selector, got, tt.want)
			}
		})
	}
}

func TestBuildBulkEntries(t *testing.T) {
	got := buildBulkEntries([]string{"BTC", "ETH"}, tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD)
	want := []tickers.Entry{
		entry("BTC", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
		entry("ETH", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildBulkEntries(...) = %v; want %v", got, want)
	}
}

func TestApplyEdit(t *testing.T) {
	base := entry("BTC", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD)

	tests := []struct {
		name string
		f    editFields
		want tickers.Entry
	}{
		{
			name: "no fields set leaves entry unchanged",
			f:    editFields{},
			want: base,
		},
		{
			name: "time only",
			f:    editFields{time: tickers.Time15Min, timeSet: true},
			want: entry("BTC", tickers.TypeCrypto, tickers.Time15Min, tickers.CurrencyUSD),
		},
		{
			name: "ticker is normalized",
			f:    editFields{ticker: " eth ", tickerSet: true},
			want: entry("ETH", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
		},
		{
			name: "type and currency together",
			f:    editFields{typ: tickers.TypeStocks, typeSet: true, currency: tickers.CurrencyEUR, currencySet: true},
			want: entry("BTC", tickers.TypeStocks, tickers.Time1Min, tickers.CurrencyEUR),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyEdit(base, tt.f); got != tt.want {
				t.Errorf("applyEdit(%v, %+v) = %v; want %v", base, tt.f, got, tt.want)
			}
		})
	}
}

func TestReorder(t *testing.T) {
	entries := []tickers.Entry{
		entry("A", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
		entry("B", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
		entry("C", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
	}

	tests := []struct {
		name      string
		from, to  int
		wantOrder []string
		wantErr   bool
	}{
		{name: "move first to last", from: 0, to: 2, wantOrder: []string{"B", "C", "A"}},
		{name: "move last to first", from: 2, to: 0, wantOrder: []string{"C", "A", "B"}},
		{name: "no-op move", from: 1, to: 1, wantOrder: []string{"A", "B", "C"}},
		{name: "from out of range", from: 3, to: 0, wantErr: true},
		{name: "to out of range", from: 0, to: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reorder(entries, tt.from, tt.to)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("reorder(from=%d, to=%d) = %v, nil; want error", tt.from, tt.to, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("reorder(from=%d, to=%d): unexpected error: %v", tt.from, tt.to, err)
			}
			gotOrder := make([]string, len(got))
			for i, e := range got {
				gotOrder[i] = e.Ticker
			}
			if !reflect.DeepEqual(gotOrder, tt.wantOrder) {
				t.Errorf("reorder(from=%d, to=%d) order = %v; want %v", tt.from, tt.to, gotOrder, tt.wantOrder)
			}
		})
	}

	t.Run("does not mutate input", func(t *testing.T) {
		before := make([]tickers.Entry, len(entries))
		copy(before, entries)
		if _, err := reorder(entries, 0, 2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(entries, before) {
			t.Errorf("reorder mutated its input: got %v; want %v", entries, before)
		}
	})
}

func TestValidateEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []tickers.Entry
		want    int
	}{
		{
			name:    "all valid, no duplicates",
			entries: []tickers.Entry{entry("BTC", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD)},
			want:    0,
		},
		{
			name:    "invalid type",
			entries: []tickers.Entry{entry("BTC", "bogus", tickers.Time1Min, tickers.CurrencyUSD)},
			want:    1,
		},
		{
			name:    "invalid time",
			entries: []tickers.Entry{entry("BTC", tickers.TypeCrypto, "bogus", tickers.CurrencyUSD)},
			want:    1,
		},
		{
			name:    "invalid currency",
			entries: []tickers.Entry{entry("BTC", tickers.TypeCrypto, tickers.Time1Min, "bogus")},
			want:    1,
		},
		{
			name: "duplicate symbols, case-insensitive",
			entries: []tickers.Entry{
				entry("btc", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
				entry("BTC", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
			},
			want: 1,
		},
		{
			name: "invalid field and duplicate both reported",
			entries: []tickers.Entry{
				entry("BTC", "bogus", tickers.Time1Min, tickers.CurrencyUSD),
				entry("BTC", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateEntries(tt.entries); len(got) != tt.want {
				t.Errorf("validateEntries(...) = %v (%d problems); want %d", got, len(got), tt.want)
			}
		})
	}
}

func TestApplyTemplatePreset(t *testing.T) {
	existing := []tickers.Entry{
		entry("BTC", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
	}
	preset := []tickers.Entry{
		entry("btc", tickers.TypeCrypto, tickers.Time5Min, tickers.CurrencyEUR),
		entry("ETH", tickers.TypeCrypto, tickers.Time5Min, tickers.CurrencyUSD),
	}

	t.Run("append dedups against existing symbols", func(t *testing.T) {
		got := applyTemplatePreset(existing, preset, false)
		want := []tickers.Entry{
			entry("BTC", tickers.TypeCrypto, tickers.Time1Min, tickers.CurrencyUSD),
			entry("ETH", tickers.TypeCrypto, tickers.Time5Min, tickers.CurrencyUSD),
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("applyTemplatePreset(append) = %v; want %v", got, want)
		}
	})

	t.Run("append leaves existing input untouched", func(t *testing.T) {
		before := make([]tickers.Entry, len(existing))
		copy(before, existing)
		applyTemplatePreset(existing, preset, false)
		if !reflect.DeepEqual(existing, before) {
			t.Errorf("applyTemplatePreset mutated existing: got %v; want %v", existing, before)
		}
	})

	t.Run("replace ignores existing entirely", func(t *testing.T) {
		got := applyTemplatePreset(existing, preset, true)
		if !reflect.DeepEqual(got, preset) {
			t.Errorf("applyTemplatePreset(replace) = %v; want %v", got, preset)
		}
	})
}
