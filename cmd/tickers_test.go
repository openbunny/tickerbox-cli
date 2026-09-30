// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func TestIsIndex(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "digits", in: "42", want: true},
		{name: "zero", in: "0", want: true},
		{name: "empty", in: "", want: false},
		{name: "ticker symbol", in: "BTC", want: false},
		{name: "mixed digits and letters", in: "4a", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIndex(tt.in); got != tt.want {
				t.Errorf("isIndex(%q) = %v; want %v", tt.in, got, tt.want)
			}
		})
	}
}

func writeTickersImportFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tickers.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestTickersRemoveConfirmation(t *testing.T) {
	origHost, origTimeout, origRetry, origJSON := resolvedHost, timeoutFlag, retryFlag, jsonFlag
	origYes, origStdin := tickersRemoveYes, confirmStdin
	t.Cleanup(func() {
		resolvedHost, timeoutFlag, retryFlag, jsonFlag = origHost, origTimeout, origRetry, origJSON
		tickersRemoveYes, confirmStdin = origYes, origStdin
	})
	timeoutFlag, retryFlag, jsonFlag = time.Second, 0, false

	getBody := `{"size":1,"types":"crypto","tickers":"BTC","times":"5min","currency":"USD"}`

	t.Run("decline exits 0 without posting", func(t *testing.T) {
		srv, posted := withTickersStub(t, getBody)
		resolvedHost, tickersRemoveYes = srv.URL, false
		confirmStdin = strings.NewReader("n\n")

		var runErr error
		stdout := captureStdout(t, func() {
			runErr = tickersRemoveCmd.RunE(tickersRemoveCmd, []string{"BTC"})
		})
		if runErr != nil {
			t.Fatalf("tickersRemoveCmd.RunE() = %v", runErr)
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
		resolvedHost, tickersRemoveYes = srv.URL, false
		confirmStdin = strings.NewReader("y\n")

		if err := tickersRemoveCmd.RunE(tickersRemoveCmd, []string{"BTC"}); err != nil {
			t.Fatalf("tickersRemoveCmd.RunE() = %v", err)
		}
		if len(*posted) == 0 {
			t.Error("device was never posted despite a confirmed removal")
		}
	})

	t.Run("--yes skips the prompt", func(t *testing.T) {
		srv, posted := withTickersStub(t, getBody)
		resolvedHost, tickersRemoveYes = srv.URL, true
		confirmStdin = strings.NewReader("")

		if err := tickersRemoveCmd.RunE(tickersRemoveCmd, []string{"BTC"}); err != nil {
			t.Fatalf("tickersRemoveCmd.RunE() = %v", err)
		}
		if len(*posted) == 0 {
			t.Error("device was never posted despite --yes")
		}
	})
}

func TestTickersImportRejectsInvalidEntries(t *testing.T) {
	var posted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			posted = true
		}
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)
	withTickersCmdTarget(t, srv.URL)

	path := writeTickersImportFile(t, `[{"type":"bogus","ticker":"BTC","time":"5min","currency":"USD"}]`)

	origInput, origYes := tickersImportInput, tickersImportYes
	t.Cleanup(func() { tickersImportInput, tickersImportYes = origInput, origYes })
	tickersImportInput, tickersImportYes = path, true

	if err := tickersImportCmd.RunE(tickersImportCmd, nil); err == nil {
		t.Fatal("tickersImportCmd.RunE() = nil error; want error for an invalid type")
	}
	if posted {
		t.Error("device was posted despite invalid input")
	}
}

func TestTickersImportReportsAccurateCount(t *testing.T) {
	var postedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			postedBody, _ = io.ReadAll(r.Body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)
	withTickersCmdTarget(t, srv.URL)

	path := writeTickersImportFile(t, `[`+
		`{"type":"crypto","ticker":"BTC","time":"5min","currency":"USD"},`+
		`{"type":"stocks","ticker":"AAPL","time":"1min","currency":"USD"}]`)

	origInput, origYes := tickersImportInput, tickersImportYes
	t.Cleanup(func() { tickersImportInput, tickersImportYes = origInput, origYes })
	tickersImportInput, tickersImportYes = path, true

	stdout := captureStdout(t, func() {
		if err := tickersImportCmd.RunE(tickersImportCmd, nil); err != nil {
			t.Fatalf("tickersImportCmd.RunE() = %v", err)
		}
	})
	if !strings.Contains(stdout, "Imported 2 entries") {
		t.Errorf("stdout = %q; want it to contain %q", stdout, "Imported 2 entries")
	}
	var gotState tickers.State
	if err := json.Unmarshal(postedBody, &gotState); err != nil {
		t.Fatalf("decode posted state: %v", err)
	}
	if gotState.Size != 2 {
		t.Errorf("posted size = %d; want 2", gotState.Size)
	}
}
