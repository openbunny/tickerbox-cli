// SPDX-License-Identifier: MIT

package tickers

import (
	"fmt"
	"slices"
	"strings"
)

type Entry struct {
	Type     string `json:"type"`
	Ticker   string `json:"ticker"`
	Time     string `json:"time"`
	Currency string `json:"currency"`
}

const (
	TypeCrypto = "crypto"
	TypeStocks = "stocks"
	TypeForex  = "forex"
)

var Types = []string{TypeCrypto, TypeStocks, TypeForex}

const (
	Time1Min  = "1min"
	Time5Min  = "5min"
	Time15Min = "15min"
)

var Times = []string{Time1Min, Time5Min, Time15Min}

const (
	CurrencyUSD = "USD"
	CurrencyEUR = "EUR"
	CurrencyGBP = "GBP"
	CurrencyCAD = "CAD"
	CurrencyAUD = "AUD"
	CurrencyJPY = "JPY"
)

var Currencies = []string{CurrencyUSD, CurrencyEUR, CurrencyGBP, CurrencyCAD, CurrencyAUD, CurrencyJPY}

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

func NormalizeTicker(ticker string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(ticker), " ", ""))
}

func ValidType(t string) bool {
	return slices.Contains(Types, t)
}

func ValidTime(t string) bool {
	return slices.Contains(Times, t)
}

func ValidCurrency(c string) bool {
	return slices.Contains(Currencies, c)
}
