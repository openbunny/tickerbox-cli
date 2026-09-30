// SPDX-License-Identifier: MIT

package template

import (
	"reflect"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

var wantPresetNames = []string{
	"ai-stocks",
	"airlines",
	"big-banks",
	"commodities-etfs",
	"crypto-majors",
	"dividend-stocks",
	"dow-industrials",
	"energy-majors",
	"ev-makers",
	"fx-majors",
	"indices",
	"mag7",
	"reits",
	"semiconductors",
	"treasuries",
}

func TestList(t *testing.T) {
	if got := List(); !reflect.DeepEqual(got, wantPresetNames) {
		t.Errorf("List() = %v; want %v", got, wantPresetNames)
	}
}

func TestAtLeast15Presets(t *testing.T) {
	if got := len(List()); got < 15 {
		t.Errorf("len(List()) = %d; want at least 15", got)
	}
}

func TestGet(t *testing.T) {
	for _, name := range wantPresetNames {
		t.Run(name, func(t *testing.T) {
			entries, ok := Get(name)
			if !ok {
				t.Fatalf("Get(%q): ok = false", name)
			}
			if len(entries) == 0 {
				t.Fatalf("Get(%q): got no entries", name)
			}
		})
	}

	t.Run("unknown preset", func(t *testing.T) {
		entries, ok := Get("does-not-exist")
		if ok || entries != nil {
			t.Errorf("Get(%q) = %v, %v; want nil, false", "does-not-exist", entries, ok)
		}
	})
}

func TestGetReturnsIndependentCopy(t *testing.T) {
	first, ok := Get("mag7")
	if !ok {
		t.Fatal("Get(mag7): ok = false")
	}
	first[0].Ticker = "MUTATED"

	second, ok := Get("mag7")
	if !ok {
		t.Fatal("Get(mag7): ok = false")
	}
	if second[0].Ticker == "MUTATED" {
		t.Error("Get returned a slice sharing storage with the built-in preset")
	}
}

func TestDescribe(t *testing.T) {
	for _, name := range wantPresetNames {
		t.Run(name, func(t *testing.T) {
			desc, ok := Describe(name)
			if !ok {
				t.Fatalf("Describe(%q): ok = false", name)
			}
			if desc == "" {
				t.Errorf("Describe(%q): got empty description", name)
			}
		})
	}

	t.Run("unknown preset", func(t *testing.T) {
		desc, ok := Describe("does-not-exist")
		if ok || desc != "" {
			t.Errorf("Describe(%q) = %q, %v; want \"\", false", "does-not-exist", desc, ok)
		}
	})
}

func TestEveryPresetEntryIsValid(t *testing.T) {
	for _, name := range List() {
		entries, ok := Get(name)
		if !ok {
			t.Fatalf("Get(%q): ok = false", name)
		}
		for _, e := range entries {
			if !tickers.ValidType(e.Type) {
				t.Errorf("preset %s: ticker %s: invalid type %q", name, e.Ticker, e.Type)
			}
			if !tickers.ValidTime(e.Time) {
				t.Errorf("preset %s: ticker %s: invalid time %q", name, e.Ticker, e.Time)
			}
			if !tickers.ValidCurrency(e.Currency) {
				t.Errorf("preset %s: ticker %s: invalid currency %q", name, e.Ticker, e.Currency)
			}
		}
	}
}

func TestPresetContents(t *testing.T) {
	tests := []struct {
		name        string
		wantTickers []string
		wantType    string
	}{
		{"mag7", []string{"AAPL", "MSFT", "GOOGL", "AMZN", "NVDA", "META", "TSLA"}, tickers.TypeStocks},
		{"indices", []string{"SPY", "QQQ", "DIA", "IWM"}, tickers.TypeStocks},
		{"crypto-majors", []string{"BTC", "ETH", "XRP", "SOL", "ADA"}, tickers.TypeCrypto},
		{"fx-majors", []string{"EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD", "USDCAD"}, tickers.TypeForex},
		{"treasuries", []string{"SHY", "IEF", "TLT", "SGOV"}, tickers.TypeStocks},
		{"semiconductors", []string{"NVDA", "AMD", "INTC", "TSM", "AVGO", "QCOM", "MU", "TXN", "ASML", "AMAT"}, tickers.TypeStocks},
		{"ai-stocks", []string{"NVDA", "MSFT", "GOOGL", "META", "AMD", "PLTR", "AVGO", "SMCI", "CRM", "ORCL"}, tickers.TypeStocks},
		{"ev-makers", []string{"TSLA", "RIVN", "LCID", "NIO", "GM", "F", "XPEV", "LI"}, tickers.TypeStocks},
		{"big-banks", []string{"JPM", "BAC", "WFC", "C", "GS", "MS", "USB", "PNC"}, tickers.TypeStocks},
		{"dividend-stocks", []string{"JNJ", "PG", "KO", "PEP", "XOM", "CVX", "MMM", "WMT", "MCD", "VZ"}, tickers.TypeStocks},
		{"dow-industrials", []string{"UNH", "GS", "HD", "CAT", "MCD", "V", "JPM", "HON", "JNJ", "CVX"}, tickers.TypeStocks},
		{"commodities-etfs", []string{"GLD", "SLV", "USO", "UNG", "DBA", "DBC"}, tickers.TypeStocks},
		{"reits", []string{"VNQ", "O", "PLD", "AMT", "SPG", "PSA", "EQIX"}, tickers.TypeStocks},
		{"energy-majors", []string{"XOM", "CVX", "COP", "SLB", "EOG", "OXY", "WMB", "PSX"}, tickers.TypeStocks},
		{"airlines", []string{"DAL", "UAL", "AAL", "LUV", "JBLU", "ALK"}, tickers.TypeStocks},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, ok := Get(tt.name)
			if !ok {
				t.Fatalf("Get(%q): ok = false", tt.name)
			}
			gotTickers := make([]string, len(entries))
			for i, e := range entries {
				gotTickers[i] = e.Ticker
				if e.Type != tt.wantType {
					t.Errorf("entry %s: type = %q; want %q", e.Ticker, e.Type, tt.wantType)
				}
				if e.Time != tickers.Time15Min {
					t.Errorf("entry %s: time = %q; want %q", e.Ticker, e.Time, tickers.Time15Min)
				}
				if e.Currency != tickers.CurrencyUSD {
					t.Errorf("entry %s: currency = %q; want %q", e.Ticker, e.Currency, tickers.CurrencyUSD)
				}
			}
			if !reflect.DeepEqual(gotTickers, tt.wantTickers) {
				t.Errorf("%s tickers = %v; want %v", tt.name, gotTickers, tt.wantTickers)
			}
		})
	}
}
