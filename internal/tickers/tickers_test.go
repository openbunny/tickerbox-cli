// SPDX-License-Identifier: MIT

package tickers

import (
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		state   State
		want    []Entry
		wantErr bool
	}{
		{
			name:  "empty state",
			state: State{},
			want:  []Entry{},
		},
		{
			name: "single crypto entry",
			state: State{
				Size:     1,
				Types:    "crypto",
				Tickers:  "BTC",
				Times:    "5min",
				Currency: "EUR",
			},
			want: []Entry{{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "EUR"}},
		},
		{
			name: "multiple mixed entries",
			state: State{
				Size:     3,
				Types:    "crypto,stocks,forex",
				Tickers:  "BTC,AAPL,EURUSD",
				Times:    "5min,1min,15min",
				Currency: "EUR,USD,USD",
			},
			want: []Entry{
				{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "EUR"},
				{Type: "stocks", Ticker: "AAPL", Time: "1min", Currency: "USD"},
				{Type: "forex", Ticker: "EURUSD", Time: "15min", Currency: "USD"},
			},
		},
		{
			name: "short currency CSV leaves trailing currency empty",
			state: State{
				Size:     2,
				Types:    "crypto,stocks",
				Tickers:  "BTC,AAPL",
				Times:    "5min,1min",
				Currency: "EUR",
			},
			want: []Entry{
				{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "EUR"},
				{Type: "stocks", Ticker: "AAPL", Time: "1min", Currency: ""},
			},
		},
		{
			name: "empty currency CSV leaves all currencies empty",
			state: State{
				Size:    1,
				Types:   "crypto",
				Tickers: "BTC",
				Times:   "5min",
			},
			want: []Entry{{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: ""}},
		},
		{
			name: "negative size errors",
			state: State{
				Size:  -1,
				Types: "crypto",
			},
			wantErr: true,
		},
		{
			name: "short types CSV errors",
			state: State{
				Size:    2,
				Types:   "crypto",
				Tickers: "BTC,AAPL",
				Times:   "5min,1min",
			},
			wantErr: true,
		},
		{
			name: "short tickers CSV errors",
			state: State{
				Size:    2,
				Types:   "crypto,stocks",
				Tickers: "BTC",
				Times:   "5min,1min",
			},
			wantErr: true,
		},
		{
			name: "short times CSV errors",
			state: State{
				Size:    2,
				Types:   "crypto,stocks",
				Tickers: "BTC,AAPL",
				Times:   "5min",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.state)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode(%+v) = %#v; want %#v", tt.state, got, tt.want)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		state State
	}{
		{
			name: "single crypto entry",
			state: State{
				Size:     1,
				Types:    "crypto",
				Tickers:  "BTC",
				Times:    "5min",
				Currency: "EUR",
			},
		},
		{
			name: "multiple mixed entries",
			state: State{
				Size:     3,
				Types:    "crypto,stocks,forex",
				Tickers:  "BTC,AAPL,EURUSD",
				Times:    "5min,1min,15min",
				Currency: "EUR,USD,USD",
			},
		},
		{
			name:  "empty state",
			state: State{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := Decode(tt.state)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			got, err := Encode(entries)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			want := map[string]any{
				"size":     tt.state.Size,
				"types":    tt.state.Types,
				"tickers":  tt.state.Tickers,
				"times":    tt.state.Times,
				"currency": tt.state.Currency,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round trip = %#v; want %#v", got, want)
			}
		})
	}
}

func TestEncodeForcesUSDForNonCrypto(t *testing.T) {
	tests := []struct {
		name         string
		entry        Entry
		wantCurrency string
	}{
		{
			name:         "crypto keeps its currency",
			entry:        Entry{Type: TypeCrypto, Ticker: "BTC", Currency: "EUR"},
			wantCurrency: "EUR",
		},
		{
			name:         "stocks forced to USD",
			entry:        Entry{Type: TypeStocks, Ticker: "AAPL", Currency: "EUR"},
			wantCurrency: CurrencyUSD,
		},
		{
			name:         "forex forced to USD",
			entry:        Entry{Type: TypeForex, Ticker: "EURJPY", Currency: "JPY"},
			wantCurrency: CurrencyUSD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode([]Entry{tt.entry})
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			if got["currency"] != tt.wantCurrency {
				t.Errorf("got currency %v; want %q", got["currency"], tt.wantCurrency)
			}
		})
	}
}

func TestEncodeRejectsEmptyTicker(t *testing.T) {
	entries := []Entry{
		{Type: "crypto", Ticker: "btc", Currency: "USD"},
		{Type: "crypto", Ticker: "  ", Currency: "USD"},
	}

	if _, err := Encode(entries); err == nil {
		t.Fatal("Encode() with a whitespace-only ticker = nil error; want error naming the entry")
	}
}

func TestEncodeAcceptsValidEntries(t *testing.T) {
	entries := []Entry{
		{Type: "crypto", Ticker: "btc", Currency: "USD"},
		{Type: "stocks", Ticker: "aapl", Currency: "USD"},
	}

	got, err := Encode(entries)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want := map[string]any{
		"size":     2,
		"types":    "crypto,stocks",
		"tickers":  "BTC,AAPL",
		"times":    ",",
		"currency": "USD,USD",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func TestEncodeRejectsCommaInField(t *testing.T) {
	tests := []struct {
		name  string
		entry Entry
	}{
		{name: "comma in type", entry: Entry{Type: "cry,pto", Ticker: "BTC", Time: "5min", Currency: "USD"}},
		{name: "comma in ticker", entry: Entry{Type: "crypto", Ticker: "BT,C", Time: "5min", Currency: "USD"}},
		{name: "comma in time", entry: Entry{Type: "crypto", Ticker: "BTC", Time: "5,min", Currency: "USD"}},
		{name: "comma in currency", entry: Entry{Type: "crypto", Ticker: "BTC", Time: "5min", Currency: "US,D"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Encode([]Entry{tt.entry}); err == nil {
				t.Fatalf("Encode(%+v) = nil error; want error rejecting the comma", tt.entry)
			}
		})
	}
}

func TestNormalizeTicker(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "lowercase becomes uppercase", in: "btc", want: "BTC"},
		{name: "internal spaces removed", in: "e u r", want: "EUR"},
		{name: "leading/trailing whitespace trimmed", in: "  aapl  ", want: "AAPL"},
		{name: "empty stays empty", in: "", want: ""},
		{name: "whitespace only becomes empty", in: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTicker(tt.in); got != tt.want {
				t.Errorf("NormalizeTicker(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidators(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) bool
		in   string
		want bool
	}{
		{name: "ValidType accepts crypto", fn: ValidType, in: "crypto", want: true},
		{name: "ValidType accepts stocks", fn: ValidType, in: "stocks", want: true},
		{name: "ValidType accepts forex", fn: ValidType, in: "forex", want: true},
		{name: "ValidType rejects unknown", fn: ValidType, in: "bonds", want: false},
		{name: "ValidType rejects empty", fn: ValidType, in: "", want: false},
		{name: "ValidTime accepts 1min", fn: ValidTime, in: "1min", want: true},
		{name: "ValidTime accepts 5min", fn: ValidTime, in: "5min", want: true},
		{name: "ValidTime accepts 15min", fn: ValidTime, in: "15min", want: true},
		{name: "ValidTime rejects unknown", fn: ValidTime, in: "1hour", want: false},
		{name: "ValidCurrency accepts USD", fn: ValidCurrency, in: "USD", want: true},
		{name: "ValidCurrency accepts JPY", fn: ValidCurrency, in: "JPY", want: true},
		{name: "ValidCurrency rejects lowercase", fn: ValidCurrency, in: "usd", want: false},
		{name: "ValidCurrency rejects unknown", fn: ValidCurrency, in: "XYZ", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.in); got != tt.want {
				t.Errorf("got %v; want %v", got, tt.want)
			}
		})
	}
}

func FuzzDecode(f *testing.F) {
	f.Add(0, "", "", "", "")
	f.Add(1, "crypto", "BTC", "5min", "EUR")
	f.Add(3, "crypto,stocks,forex", "BTC,AAPL,EURUSD", "5min,1min,15min", "EUR,USD")
	f.Add(-1, "crypto", "BTC", "5min", "EUR")
	f.Add(5, "crypto", "BTC", "5min", "EUR")
	f.Add(2, "crypto,stocks", "BTC,AAPL", "5min,1min", "")

	f.Fuzz(func(t *testing.T, size int, types, tickers, tms, currency string) {
		state := State{Size: size, Types: types, Tickers: tickers, Times: tms, Currency: currency}
		entries, err := Decode(state)
		if err != nil {
			if entries != nil {
				t.Fatalf("Decode returned entries %#v alongside error %v", entries, err)
			}
			return
		}
		if len(entries) != size {
			t.Fatalf("Decode returned %d entries; want size %d", len(entries), size)
		}
		if _, err := Encode(entries); err != nil {
			return
		}
	})
}
