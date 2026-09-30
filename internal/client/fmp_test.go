// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func withFMPServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFMPVerifyValidSymbol(t *testing.T) {
	srv := withFMPServer(t, http.StatusOK, `[{"symbol":"AAPL"}]`)
	f := &FMPClient{c: New(srv.URL+"/", time.Second)}

	known, err := f.Verify(context.Background(), "AAPL", "key")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !known {
		t.Error("Verify() = false, want true for a non-empty result array")
	}
}

func TestFMPVerifyUnknownSymbol(t *testing.T) {
	srv := withFMPServer(t, http.StatusOK, `[]`)
	f := &FMPClient{c: New(srv.URL+"/", time.Second)}

	known, err := f.Verify(context.Background(), "ZZZZINVALIDTICKER123", "key")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if known {
		t.Error("Verify() = true, want false for an empty result array")
	}
}

// secretAPIKey is a distinctive value so a test can assert it never surfaces in
// a formatted error, rather than asserting on a generic string like "key" that
// could appear in error text incidentally.
const secretAPIKey = "sk-do-not-leak-9f3a2b1c"

func TestFMPVerifyUnauthorized(t *testing.T) {
	srv := withFMPServer(t, http.StatusUnauthorized, `{"error":"invalid key"}`)
	f := &FMPClient{c: New(srv.URL+"/", time.Second)}

	_, err := f.Verify(context.Background(), "AAPL", secretAPIKey)
	if err == nil {
		t.Fatal("Verify() error = nil, want an error on HTTP 401")
	}
	if strings.Contains(err.Error(), secretAPIKey) {
		t.Errorf("error = %q; must not contain the API key", err.Error())
	}
}

func TestFMPVerifyServerError(t *testing.T) {
	srv := withFMPServer(t, http.StatusInternalServerError, `boom`)
	f := &FMPClient{c: New(srv.URL+"/", time.Second)}

	_, err := f.Verify(context.Background(), "AAPL", secretAPIKey)
	if err == nil {
		t.Fatal("Verify() error = nil, want an error on HTTP 500")
	}
	if strings.Contains(err.Error(), secretAPIKey) {
		t.Errorf("error = %q; must not contain the API key", err.Error())
	}
}

func TestFMPVerifyTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"symbol":"AAPL"}]`))
	}))
	t.Cleanup(srv.Close)
	f := &FMPClient{c: New(srv.URL+"/", 20*time.Millisecond)}

	_, err := f.Verify(context.Background(), "AAPL", secretAPIKey)
	if err == nil {
		t.Fatal("Verify() error = nil, want an error on timeout")
	}
	if strings.Contains(err.Error(), secretAPIKey) {
		t.Errorf("error = %q; must not contain the API key", err.Error())
	}
}

func TestFMPVerifyConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	f := &FMPClient{c: New(url+"/", time.Second)}

	_, err := f.Verify(context.Background(), "AAPL", secretAPIKey)
	if err == nil {
		t.Fatal("Verify() error = nil, want an error on connection refused")
	}
	if strings.Contains(err.Error(), secretAPIKey) {
		t.Errorf("error = %q; must not contain the API key", err.Error())
	}
}

func TestFMPUnreachableNeverEqualsUnknown(t *testing.T) {
	unreachableSrv := withFMPServer(t, http.StatusInternalServerError, `boom`)
	unreachable := &FMPClient{c: New(unreachableSrv.URL+"/", time.Second)}
	_, unreachableErr := unreachable.Verify(context.Background(), "AAPL", "key")
	if unreachableErr == nil {
		t.Fatal("expected an unreachable error")
	}

	unknownSrv := withFMPServer(t, http.StatusOK, `[]`)
	unknown := &FMPClient{c: New(unknownSrv.URL+"/", time.Second)}
	known, unknownErr := unknown.Verify(context.Background(), "ZZZZINVALIDTICKER123", "key")
	if unknownErr != nil {
		t.Fatalf("unexpected error for a 200/[] response: %v", unknownErr)
	}
	if known {
		t.Fatal("expected known = false for a 200/[] response")
	}
	if strings.Contains(unreachableErr.Error(), "[]") {
		t.Error("unreachable error text unexpectedly resembles the unknown-symbol case")
	}
}

func TestNewFMPUsesFMPBaseURL(t *testing.T) {
	orig := FMPBaseURL
	t.Cleanup(func() { FMPBaseURL = orig })
	FMPBaseURL = "http://example.invalid/"

	f := NewFMP(time.Second)
	if f.c.base != FMPBaseURL {
		t.Errorf("NewFMP() base = %q, want %q", f.c.base, FMPBaseURL)
	}
}
