// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/config"
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

// Targets port 9 (discard): unreachable on a real network and denied fast under the
// sandbox either way, so a passing test proves validation ran before any request.
func TestTickersAddBulkRejectsInvalidTypeWithoutContactingDevice(t *testing.T) {
	origHost, origTimeout, origRetry, origJSON := resolvedHost, timeoutFlag, retryFlag, jsonFlag
	t.Cleanup(func() {
		resolvedHost, timeoutFlag, retryFlag, jsonFlag = origHost, origTimeout, origRetry, origJSON
	})
	resolvedHost, timeoutFlag, retryFlag, jsonFlag = "http://127.0.0.1:9", 50*time.Millisecond, 0, false

	origType, origTime, origCurrency := tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency
	t.Cleanup(func() {
		tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = origType, origTime, origCurrency
	})
	tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = "crytpo", "5min", "USD"

	err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"BTC"})
	if err == nil {
		t.Fatal("tickersAddBulkCmd.RunE() = nil error; want error for invalid --type")
	}
	if !strings.Contains(err.Error(), `invalid type "crytpo"`) {
		t.Errorf("error = %q; want it to name the invalid type, not a network failure", err.Error())
	}
}

func TestTickersAddBulkAcceptsValidEntries(t *testing.T) {
	srv, posted := withTickersStub(t, `{"size":0}`)
	withTickersCmdTarget(t, srv.URL)

	origType, origTime, origCurrency := tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency
	origNoVerify := tickersAddBulkNoVerify
	t.Cleanup(func() {
		tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = origType, origTime, origCurrency
		tickersAddBulkNoVerify = origNoVerify
	})
	tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = "crypto", "5min", "USD"
	tickersAddBulkNoVerify = true

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

// TestTickersAddBulkStaysConfirmationFree proves no confirm() read was inserted
// into the add path: with stdin closed, a valid add still succeeds.
func TestTickersAddBulkStaysConfirmationFree(t *testing.T) {
	srv, posted := withTickersStub(t, `{"size":0}`)
	withTickersCmdTarget(t, srv.URL)

	origType, origTime, origCurrency, origNoVerify, origStdin := tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency, tickersAddBulkNoVerify, confirmStdin
	t.Cleanup(func() {
		tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = origType, origTime, origCurrency
		tickersAddBulkNoVerify, confirmStdin = origNoVerify, origStdin
	})
	tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency = "crypto", "5min", "USD"
	tickersAddBulkNoVerify = true
	confirmStdin = strings.NewReader("")

	if err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"BTC"}); err != nil {
		t.Fatalf("tickersAddBulkCmd.RunE() (closed stdin) = %v", err)
	}
	if len(*posted) == 0 {
		t.Error("device was never posted despite a valid add")
	}
}

// TestTickersMoveStaysConfirmationFree guards requirement C's scope: an additive
// command (move) must gain neither a confirm() call nor a --yes flag.
func TestTickersMoveStaysConfirmationFree(t *testing.T) {
	srv, posted := withTickersStub(t, `{"size":2,"types":"crypto,stocks","tickers":"BTC,AAPL","times":"5min,1min","currency":"USD,USD"}`)
	withTickersCmdTarget(t, srv.URL)

	origStdin := confirmStdin
	t.Cleanup(func() { confirmStdin = origStdin })
	confirmStdin = strings.NewReader("")

	if err := tickersMoveCmd.RunE(tickersMoveCmd, []string{"0", "1"}); err != nil {
		t.Fatalf("tickersMoveCmd.RunE() (closed stdin) = %v", err)
	}
	if len(*posted) == 0 {
		t.Error("device was never posted despite a valid move")
	}
	if tickersMoveCmd.Flags().Lookup("yes") != nil {
		t.Error("tickersMoveCmd unexpectedly has a --yes flag")
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

// secretAPIKey is a distinctive value so a test can assert it never surfaces in
// a formatted error, rather than asserting on a generic string that could
// appear in error text incidentally.
const secretAPIKey = "sk-do-not-leak-9f3a2b1c"

func withFMPStub(t *testing.T, status int, body string) (*httptest.Server, *int) {
	t.Helper()
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	origBase := client.FMPBaseURL
	t.Cleanup(func() { client.FMPBaseURL = origBase })
	client.FMPBaseURL = srv.URL + "/"
	return srv, &requests
}

func tickersAddBulkVerifyFixture(t *testing.T) (*[]byte, func()) {
	t.Helper()
	deviceSrv, posted := withTickersStub(t, `{"size":0}`)
	withTickersCmdTarget(t, deviceSrv.URL)

	origType, origTime, origCurrency, origNoVerify := tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency, tickersAddBulkNoVerify
	tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency, tickersAddBulkNoVerify = "crypto", "5min", "USD", false
	cleanup := func() {
		tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency, tickersAddBulkNoVerify = origType, origTime, origCurrency, origNoVerify
	}
	return posted, cleanup
}

func TestTickersAddBulkVerifyValid(t *testing.T) {
	posted, cleanup := tickersAddBulkVerifyFixture(t)
	defer cleanup()
	_, fmpRequests := withFMPStub(t, http.StatusOK, `[{"symbol":"AAPL"}]`)
	t.Setenv(config.EnvFMPAPIKey, "test-key")

	if err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"AAPL"}); err != nil {
		t.Fatalf("tickersAddBulkCmd.RunE() = %v", err)
	}
	if len(*posted) == 0 {
		t.Error("device was never posted despite a verified symbol")
	}
	if *fmpRequests != 1 {
		t.Errorf("FMP requests = %d, want 1", *fmpRequests)
	}
}

func TestTickersAddBulkVerifyUnknownSymbol(t *testing.T) {
	posted, cleanup := tickersAddBulkVerifyFixture(t)
	defer cleanup()
	withFMPStub(t, http.StatusOK, `[]`)
	t.Setenv(config.EnvFMPAPIKey, "test-key")

	err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"ZZZZINVALIDTICKER123"})
	if err == nil {
		t.Fatal("tickersAddBulkCmd.RunE() = nil error; want error for an unknown symbol")
	}
	want := `ticker "ZZZZINVALIDTICKER123": not found on Financial Modeling Prep — check the symbol (e.g. AAPL, BTC, EURUSD) or pass --no-verify to add it unchecked`
	if err.Error() != want {
		t.Errorf("error = %q; want %q", err.Error(), want)
	}
	if len(*posted) != 0 {
		t.Error("device was posted despite an unknown symbol")
	}
}

func TestTickersAddBulkVerifyNoVerifySkipsFMP(t *testing.T) {
	posted, cleanup := tickersAddBulkVerifyFixture(t)
	defer cleanup()
	tickersAddBulkNoVerify = true
	// No FMP stub is started; client.FMPBaseURL still points at the real host, but a
	// request there would fail the sandboxed test run if one were ever attempted.

	if err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"AAPL"}); err != nil {
		t.Fatalf("tickersAddBulkCmd.RunE() = %v", err)
	}
	if len(*posted) == 0 {
		t.Error("device was never posted despite --no-verify")
	}
}

func TestTickersAddBulkVerifyUnreachable(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T)
	}{
		{
			name: "HTTP 500",
			setup: func(t *testing.T) {
				withFMPStub(t, http.StatusInternalServerError, `boom`)
			},
		},
		{
			name: "HTTP 401",
			setup: func(t *testing.T) {
				withFMPStub(t, http.StatusUnauthorized, `{"error":"invalid key"}`)
			},
		},
		{
			name: "connection refused",
			setup: func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				url := srv.URL
				srv.Close()
				origBase := client.FMPBaseURL
				t.Cleanup(func() { client.FMPBaseURL = origBase })
				client.FMPBaseURL = url + "/"
			},
		},
		{
			name: "timeout",
			setup: func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(200 * time.Millisecond)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[{"symbol":"AAPL"}]`))
				}))
				t.Cleanup(srv.Close)
				origBase := client.FMPBaseURL
				t.Cleanup(func() { client.FMPBaseURL = origBase })
				client.FMPBaseURL = srv.URL + "/"

				origTimeout := fmpVerifyTimeout
				t.Cleanup(func() { fmpVerifyTimeout = origTimeout })
				fmpVerifyTimeout = 20 * time.Millisecond
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posted, cleanup := tickersAddBulkVerifyFixture(t)
			defer cleanup()
			t.Setenv(config.EnvFMPAPIKey, secretAPIKey)
			tt.setup(t)

			err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"AAPL"})
			if err == nil {
				t.Fatal("tickersAddBulkCmd.RunE() = nil error; want error for an unreachable FMP")
			}
			if !strings.Contains(err.Error(), "Financial Modeling Prep unreachable") {
				t.Errorf("error = %q; want it to name FMP as unreachable", err.Error())
			}
			if strings.Contains(err.Error(), "not found on Financial Modeling Prep") {
				t.Errorf("error = %q; unreachable must never read as unknown-symbol", err.Error())
			}
			if strings.Contains(err.Error(), secretAPIKey) {
				t.Errorf("error = %q; must not contain the API key", err.Error())
			}
			if len(*posted) != 0 {
				t.Error("device was posted despite an unreachable FMP")
			}
		})
	}
}

func TestTickersAddBulkVerifyMissingAPIKey(t *testing.T) {
	posted, cleanup := tickersAddBulkVerifyFixture(t)
	defer cleanup()
	_, fmpRequests := withFMPStub(t, http.StatusOK, `[{"symbol":"AAPL"}]`)
	t.Setenv(config.EnvFMPAPIKey, "")
	withCmdTempHome(t)

	// Multiple symbols: the missing-key error must name none of them, guarding
	// against a regression to naming one symbol out of several.
	err := tickersAddBulkCmd.RunE(tickersAddBulkCmd, []string{"AAPL", "BTC", "EURUSD"})
	if err == nil {
		t.Fatal("tickersAddBulkCmd.RunE() = nil error; want error for a missing API key")
	}
	want := `verify tickers: no Financial Modeling Prep API key configured — set TICKERBOX_FMP_API_KEY, or pass --no-verify to add without verification`
	if err.Error() != want {
		t.Errorf("error = %q; want %q", err.Error(), want)
	}
	for _, sym := range []string{"AAPL", "BTC", "EURUSD"} {
		if strings.Contains(err.Error(), sym) {
			t.Errorf("error = %q; must not name any specific ticker (found %q)", err.Error(), sym)
		}
	}
	if *fmpRequests != 0 {
		t.Errorf("FMP requests = %d, want 0 before any request is sent", *fmpRequests)
	}
	if len(*posted) != 0 {
		t.Error("device was posted despite a missing API key")
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

	origTime, origYes := tickersEditTime, tickersEditYes
	t.Cleanup(func() { tickersEditTime, tickersEditYes = origTime, origYes })
	tickersEditTime = "1min"
	tickersEditYes = true

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

func TestTickersEditConfirmation(t *testing.T) {
	origYes, origStdin := tickersEditYes, confirmStdin
	t.Cleanup(func() { tickersEditYes, confirmStdin = origYes, origStdin })

	getBody := `{"size":1,"types":"crypto","tickers":"BTC","times":"5min","currency":"USD"}`

	t.Run("decline exits 0 without posting", func(t *testing.T) {
		srv, posted := withTickersStub(t, getBody)
		withTickersCmdTarget(t, srv.URL)
		tickersEditYes = false
		confirmStdin = strings.NewReader("n\n")

		c := tickersEditCmdFixture()
		var runErr error
		stdout := captureStdout(t, func() {
			runErr = tickersEditCmd.RunE(c, []string{"BTC"})
		})
		if runErr != nil {
			t.Fatalf("tickersEditCmd.RunE() = %v", runErr)
		}
		if !strings.Contains(stdout, "aborted") {
			t.Errorf("stdout = %q; want it to contain %q", stdout, "aborted")
		}
		if len(*posted) != 0 {
			t.Error("device was posted despite a declined confirmation")
		}
	})

	t.Run("confirm proceeds", func(t *testing.T) {
		srv, posted := withTickersStub(t, getBody)
		withTickersCmdTarget(t, srv.URL)
		tickersEditYes = false
		confirmStdin = strings.NewReader("y\n")

		c := tickersEditCmdFixture()
		if err := tickersEditCmd.RunE(c, []string{"BTC"}); err != nil {
			t.Fatalf("tickersEditCmd.RunE() = %v", err)
		}
		if len(*posted) == 0 {
			t.Error("device was never posted despite a confirmed edit")
		}
	})
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

	origReplace, origYes := tickersTemplateReplace, tickersTemplateApplyYes
	t.Cleanup(func() { tickersTemplateReplace, tickersTemplateApplyYes = origReplace, origYes })
	tickersTemplateReplace = true
	tickersTemplateApplyYes = true

	if err := tickersTemplateApplyCmd.RunE(tickersTemplateApplyCmd, []string{names[0]}); err != nil {
		t.Fatalf("tickersTemplateApplyCmd.RunE() = %v", err)
	}
	if len(*posted) == 0 {
		t.Error("device was never posted")
	}
}

func TestTickersTemplateApplyReplaceConfirmation(t *testing.T) {
	names := template.List()
	if len(names) == 0 {
		t.Skip("no templates registered")
	}

	origReplace, origYes, origStdin := tickersTemplateReplace, tickersTemplateApplyYes, confirmStdin
	t.Cleanup(func() {
		tickersTemplateReplace, tickersTemplateApplyYes, confirmStdin = origReplace, origYes, origStdin
	})
	tickersTemplateReplace = true

	t.Run("decline exits 0 without posting", func(t *testing.T) {
		srv, posted := withTickersStub(t, `{"size":0}`)
		withTickersCmdTarget(t, srv.URL)
		tickersTemplateApplyYes = false
		confirmStdin = strings.NewReader("n\n")

		var runErr error
		stdout := captureStdout(t, func() {
			runErr = tickersTemplateApplyCmd.RunE(tickersTemplateApplyCmd, []string{names[0]})
		})
		if runErr != nil {
			t.Fatalf("tickersTemplateApplyCmd.RunE() = %v", runErr)
		}
		if !strings.Contains(stdout, "aborted") {
			t.Errorf("stdout = %q; want it to contain %q", stdout, "aborted")
		}
		if len(*posted) != 0 {
			t.Error("device was posted despite a declined confirmation")
		}
	})

	t.Run("confirm proceeds", func(t *testing.T) {
		srv, posted := withTickersStub(t, `{"size":0}`)
		withTickersCmdTarget(t, srv.URL)
		tickersTemplateApplyYes = false
		confirmStdin = strings.NewReader("y\n")

		if err := tickersTemplateApplyCmd.RunE(tickersTemplateApplyCmd, []string{names[0]}); err != nil {
			t.Fatalf("tickersTemplateApplyCmd.RunE() = %v", err)
		}
		if len(*posted) == 0 {
			t.Error("device was never posted despite a confirmed replace")
		}
	})
}

func TestTickersTemplateApplyAppendStaysConfirmationFree(t *testing.T) {
	names := template.List()
	if len(names) == 0 {
		t.Skip("no templates registered")
	}
	srv, posted := withTickersStub(t, `{"size":0}`)
	withTickersCmdTarget(t, srv.URL)

	origReplace, origYes, origStdin := tickersTemplateReplace, tickersTemplateApplyYes, confirmStdin
	t.Cleanup(func() {
		tickersTemplateReplace, tickersTemplateApplyYes, confirmStdin = origReplace, origYes, origStdin
	})
	tickersTemplateReplace = false
	tickersTemplateApplyYes = false
	confirmStdin = strings.NewReader("")

	if err := tickersTemplateApplyCmd.RunE(tickersTemplateApplyCmd, []string{names[0]}); err != nil {
		t.Fatalf("tickersTemplateApplyCmd.RunE() (append, closed stdin) = %v", err)
	}
	if len(*posted) == 0 {
		t.Error("device was never posted despite a valid append")
	}
}
