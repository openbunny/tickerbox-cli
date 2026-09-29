// Package tickers implements the TickerBox device's ticker/asset list
// domain: the coinSetupState wire format and the rules for converting it
// to and from a list of Entry values.
package tickers

import (
	"fmt"
	"strings"
)

// Entry is one configured ticker.
type Entry struct {
	Type     string `json:"type"`
	Ticker   string `json:"ticker"`
	Time     string `json:"time"`
	Currency string `json:"currency"`
}

// Ticker types the device accepts.
const (
	TypeCrypto = "crypto"
	TypeStocks = "stocks"
	TypeForex  = "forex"
)

// Types lists the ticker types the device accepts.
var Types = []string{TypeCrypto, TypeStocks, TypeForex}

// Update intervals the device accepts.
const (
	Time1Min  = "1min"
	Time5Min  = "5min"
	Time15Min = "15min"
)

// Times lists the update intervals the device accepts.
var Times = []string{Time1Min, Time5Min, Time15Min}

// Currencies the device accepts.
const (
	CurrencyUSD = "USD"
	CurrencyEUR = "EUR"
	CurrencyGBP = "GBP"
	CurrencyCAD = "CAD"
	CurrencyAUD = "AUD"
	CurrencyJPY = "JPY"
)

// Currencies lists the currencies the device accepts.
var Currencies = []string{CurrencyUSD, CurrencyEUR, CurrencyGBP, CurrencyCAD, CurrencyAUD, CurrencyJPY}

// State is the device's coinSetupState wire format: a count plus four
// comma-joined fields, zipped positionally up to Size.
type State struct {
	Size     int    `json:"size"`
	Types    string `json:"types"`
	Tickers  string `json:"tickers"`
	Times    string `json:"times"`
	Currency string `json:"currency"`
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// Decode parses a coinSetupState response into Entry values, zipping
// Types, Tickers, Times and Currency positionally up to Size. Types,
// Tickers and Times must each carry at least Size comma-separated
// elements; Currency may carry fewer, and positions past its end decode
// to an empty currency.
func Decode(state State) ([]Entry, error) {
	if state.Size < 0 {
		return nil, fmt.Errorf("coinSetupState: negative size %d", state.Size)
	}

	types := splitCSV(state.Types)
	tickers := splitCSV(state.Tickers)
	times := splitCSV(state.Times)
	currencies := splitCSV(state.Currency)

	if len(types) < state.Size {
		return nil, fmt.Errorf("coinSetupState: types has %d entries, want at least %d", len(types), state.Size)
	}
	if len(tickers) < state.Size {
		return nil, fmt.Errorf("coinSetupState: tickers has %d entries, want at least %d", len(tickers), state.Size)
	}
	if len(times) < state.Size {
		return nil, fmt.Errorf("coinSetupState: times has %d entries, want at least %d", len(times), state.Size)
	}

	entries := make([]Entry, state.Size)
	for i := range entries {
		entries[i].Type = types[i]
		entries[i].Ticker = tickers[i]
		entries[i].Time = times[i]
		if i < len(currencies) {
			entries[i].Currency = currencies[i]
		}
	}
	return entries, nil
}

// Encode builds a coinSetupState request body from entries: each ticker
// is upper-cased and trimmed of whitespace, entries left with an empty
// ticker are dropped, non-crypto entries have their currency forced to
// USD, and size reflects the resulting entry count.
func Encode(entries []Entry) map[string]any {
	types := make([]string, 0, len(entries))
	tickers := make([]string, 0, len(entries))
	times := make([]string, 0, len(entries))
	currencies := make([]string, 0, len(entries))

	for _, e := range entries {
		ticker := NormalizeTicker(e.Ticker)
		if ticker == "" {
			continue
		}
		currency := e.Currency
		if e.Type != TypeCrypto {
			currency = CurrencyUSD
		}
		types = append(types, e.Type)
		tickers = append(tickers, ticker)
		times = append(times, e.Time)
		currencies = append(currencies, currency)
	}

	return map[string]any{
		"size":     len(tickers),
		"types":    strings.Join(types, ","),
		"tickers":  strings.Join(tickers, ","),
		"times":    strings.Join(times, ","),
		"currency": strings.Join(currencies, ","),
	}
}

// NormalizeTicker upper-cases ticker and strips its whitespace, the form
// the device expects and the form entries are matched against.
func NormalizeTicker(ticker string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(ticker), " ", ""))
}

func contains(values []string, v string) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}

// ValidType reports whether t is one of the ticker types the device accepts.
func ValidType(t string) bool {
	return contains(Types, t)
}

// ValidTime reports whether t is one of the update intervals the device accepts.
func ValidTime(t string) bool {
	return contains(Times, t)
}

// ValidCurrency reports whether c is one of the currencies the device accepts.
func ValidCurrency(c string) bool {
	return contains(Currencies, c)
}
