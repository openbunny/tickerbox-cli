// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"net/url"
	"time"
)

// FMPBaseURL is a var, not a const, so a test can point NewFMP at an httptest
// server instead of the real Financial Modeling Prep host.
var FMPBaseURL = "https://financialmodelingprep.com/stable/"

type FMPClient struct {
	c *Client
}

func NewFMP(timeout time.Duration) *FMPClient {
	return &FMPClient{c: New(FMPBaseURL, timeout)}
}

type fmpQuote struct {
	Symbol string `json:"symbol"`
}

// Verify reports whether symbol is a known FMP ticker: true for a 200 response
// with a non-empty JSON array, false for a 200 with []. A non-nil error means
// verification itself failed (401/429/5xx, a transport error, or a timeout) and
// says nothing about whether the symbol exists — the caller must not treat it
// as unknown. The request path carries apiKey in its query string, so it is
// never used as the error label — GetLabeled reports a key-free label instead,
// keeping the key out of every formatted or printed error, including a
// transport failure or timeout that wraps the request path verbatim.
//
// The empty-array-on-miss convention this reads is inherited vendor/community
// behavior, not confirmed against a live API key: FMP's public demo key returns
// 401 for both a real and a fabricated symbol, so the true/false split above is
// untested against a real response and needs a smoke test with a provisioned
// key before being trusted.
func (f *FMPClient) Verify(ctx context.Context, symbol, apiKey string) (bool, error) {
	label := "quote?symbol=" + url.QueryEscape(symbol)
	path := label + "&apikey=" + url.QueryEscape(apiKey)
	var quotes []fmpQuote
	if err := f.c.GetLabeled(ctx, path, label, &quotes); err != nil {
		return false, err
	}
	return len(quotes) > 0, nil
}
