// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/template"
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

func withTickersCmdTarget(t *testing.T, host string) {
	t.Helper()
	origHost, origTimeout, origRetry, origJSON := resolvedHost, timeoutFlag, retryFlag, jsonFlag
	t.Cleanup(func() {
		resolvedHost, timeoutFlag, retryFlag, jsonFlag = origHost, origTimeout, origRetry, origJSON
	})
	resolvedHost, timeoutFlag, retryFlag, jsonFlag = host, time.Second, 0, false
}

func withTickersStub(t *testing.T, getBody string) (*httptest.Server, *[]byte) {
	t.Helper()
	var posted []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			posted, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{})
			return
		}
		_, _ = w.Write([]byte(getBody))
	}))
	t.Cleanup(srv.Close)
	return srv, &posted
}

func TestTickersAddBulkRejectsInvalidType(t *testing.T) {
	srv, posted := withTickersStub(t, `{"size":0}`)
	withTickersCmdTarget(t, srv.URL)

	origType, origTime, origCurrency := tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency
	t.Cleanup(func() {
		tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = origType, origTime, origCurrency
	})
	tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = "bogus", "5min", "USD"

	if err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"BTC"}); err == nil {
		t.Fatal("tickersAddBulkCmd.RunE() = nil error; want error for invalid type")
	}
	if len(*posted) != 0 {
		t.Error("device was posted despite invalid type")
	}
}

func TestTickersAddBulkAcceptsValidEntries(t *testing.T) {
	srv, posted := withTickersStub(t, `{"size":0}`)
	withTickersCmdTarget(t, srv.URL)

	origType, origTime, origCurrency := tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency
	t.Cleanup(func() {
		tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = origType, origTime, origCurrency
	})
	tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = "crypto", "5min", "USD"

	if err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"BTC"}); err != nil {
		t.Fatalf("tickersAddBulkCmd.RunE() = %v", err)
	}
	var gotState tickers.State
	if err := json.Unmarshal(*posted, &gotState); err != nil {
		t.Fatalf("decode posted state: %v", err)
	}
	if gotState.Tickers != "BTC" {
		t.Errorf("posted tickers = %q; want %q", gotState.Tickers, "BTC")
	}
}

func TestTickersAddBulkRejectsCommaInTicker(t *testing.T) {
	srv, posted := withTickersStub(t, `{"size":0}`)
	withTickersCmdTarget(t, srv.URL)

	origType, origTime, origCurrency := tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency
	t.Cleanup(func() {
		tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = origType, origTime, origCurrency
	})
	tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = "crypto", "5min", "USD"

	if err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"BT,C"}); err == nil {
		t.Fatal("tickersAddBulkCmd.RunE() = nil error; want error for a comma in the ticker")
	}
	if len(*posted) != 0 {
		t.Error("device was posted despite the comma-corrupted ticker")
	}
}

func tickersEditCmdFixture() *cobra.Command {
	c := &cobra.Command{Use: "edit"}
	c.Flags().String("ticker", "", "")
	c.Flags().String("type", "", "")
	c.Flags().String("time", "", "")
	c.Flags().String("currency", "", "")
	return c
}

func TestTickersEditRejectsInvalidTime(t *testing.T) {
	srv, posted := withTickersStub(t, `{"size":1,"types":"crypto","tickers":"BTC","times":"5min","currency":"USD"}`)
	withTickersCmdTarget(t, srv.URL)

	origTime := tickersEditTime
	t.Cleanup(func() { tickersEditTime = origTime })
	tickersEditTime = "bogus"

	c := tickersEditCmdFixture()
	if err := c.Flags().Set("time", "bogus"); err != nil {
		t.Fatalf("set time flag: %v", err)
	}

	if err := tickersEditCmd.RunE(c, []string{"BTC"}); err == nil {
		t.Fatal("tickersEditCmd.RunE() = nil error; want error for invalid time")
	}
	if len(*posted) != 0 {
		t.Error("device was posted despite invalid time")
	}
}

func TestTickersEditAcceptsValidTime(t *testing.T) {
	srv, posted := withTickersStub(t, `{"size":1,"types":"crypto","tickers":"BTC","times":"5min","currency":"USD"}`)
	withTickersCmdTarget(t, srv.URL)

	origTime := tickersEditTime
	t.Cleanup(func() { tickersEditTime = origTime })
	tickersEditTime = "1min"

	c := tickersEditCmdFixture()
	if err := c.Flags().Set("time", "1min"); err != nil {
		t.Fatalf("set time flag: %v", err)
	}

	if err := tickersEditCmd.RunE(c, []string{"BTC"}); err != nil {
		t.Fatalf("tickersEditCmd.RunE() = %v", err)
	}
	var gotState tickers.State
	if err := json.Unmarshal(*posted, &gotState); err != nil {
		t.Fatalf("decode posted state: %v", err)
	}
	if gotState.Times != "1min" {
		t.Errorf("posted times = %q; want %q", gotState.Times, "1min")
	}
}

func TestTickersTemplateApplyRejectsWhenExistingEntryInvalid(t *testing.T) {
	names := template.List()
	if len(names) == 0 {
		t.Skip("no templates registered")
	}
	srv, posted := withTickersStub(t, `{"size":1,"types":"bogus","tickers":"XYZ","times":"5min","currency":"USD"}`)
	withTickersCmdTarget(t, srv.URL)

	origReplace := tickersTemplateReplace
	t.Cleanup(func() { tickersTemplateReplace = origReplace })
	tickersTemplateReplace = false

	if err := tickersTemplateApplyCmd.RunE(tickersTemplateApplyCmd, []string{names[0]}); err == nil {
		t.Fatal("tickersTemplateApplyCmd.RunE() = nil error; want error for the pre-existing invalid entry")
	}
	if len(*posted) != 0 {
		t.Error("device was posted despite an invalid existing entry")
	}
}

func TestTickersTemplateApplyAcceptsValidReplace(t *testing.T) {
	names := template.List()
	if len(names) == 0 {
		t.Skip("no templates registered")
	}
	srv, posted := withTickersStub(t, `{"size":0}`)
	withTickersCmdTarget(t, srv.URL)

	origReplace := tickersTemplateReplace
	t.Cleanup(func() { tickersTemplateReplace = origReplace })
	tickersTemplateReplace = true

	if err := tickersTemplateApplyCmd.RunE(tickersTemplateApplyCmd, []string{names[0]}); err != nil {
		t.Fatalf("tickersTemplateApplyCmd.RunE() = %v", err)
	}
	if len(*posted) == 0 {
		t.Error("device was never posted")
	}
}
