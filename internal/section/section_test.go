// SPDX-License-Identifier: MIT

package section

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func newTestServer(t *testing.T, bodies map[string]string) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var requested []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]
		mu.Lock()
		requested = append(requested, path)
		mu.Unlock()
		body, ok := bodies[path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	return srv, &requested
}

func TestCaptureDefaultInclude(t *testing.T) {
	bodies := map[string]string{
		pathCoinSetupState:  `{"size":1,"types":"stocks","tickers":"AAPL","times":"15min","currency":"USD"}`,
		pathSettingsState:   `{"brightness":200}`,
		pathClockSetupState: `{"enabled":true}`,
		pathNTPSettings:     `{"server":"pool.ntp.org"}`,
	}
	srv, requested := newTestServer(t, bodies)
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	s, err := Capture(context.Background(), c, DefaultInclude, false)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	sort.Strings(*requested)
	want := []string{pathClockSetupState, pathCoinSetupState, pathNTPSettings, pathSettingsState}
	if !reflect.DeepEqual(*requested, want) {
		t.Errorf("requested paths = %v; want %v", *requested, want)
	}
	if s.Wifi != nil || s.AP != nil {
		t.Errorf("expected wifi and ap unset with DefaultInclude, got wifi=%v ap=%v", s.Wifi, s.AP)
	}
	if len(s.Tickers) != 1 || s.Tickers[0].Ticker != "AAPL" {
		t.Errorf("got tickers %+v; want one AAPL entry", s.Tickers)
	}
	if s.Display["brightness"] != float64(200) {
		t.Errorf("got display %v; want brightness 200", s.Display)
	}
}

func TestCaptureStripsSecrets(t *testing.T) {
	bodies := map[string]string{
		pathWifiSettings: `{"ssid":"home","password":"hunter2"}`,
		pathAPSettings:   `{"ssid":"box-ap","secretKey":"topsecret"}`,
	}

	tests := []struct {
		name        string
		withSecrets bool
		wantWifi    map[string]any
		wantAP      map[string]any
	}{
		{
			name:        "secrets stripped by default",
			withSecrets: false,
			wantWifi:    map[string]any{"ssid": "home"},
			wantAP:      map[string]any{"ssid": "box-ap"},
		},
		{
			name:        "secrets kept when requested",
			withSecrets: true,
			wantWifi:    map[string]any{"ssid": "home", "password": "hunter2"},
			wantAP:      map[string]any{"ssid": "box-ap", "secretKey": "topsecret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := newTestServer(t, bodies)
			defer srv.Close()

			c := client.New(srv.URL+"/", 0)
			s, err := Capture(context.Background(), c, []string{SectionWifi, SectionAP}, tt.withSecrets)
			if err != nil {
				t.Fatalf("Capture: %v", err)
			}
			if !reflect.DeepEqual(s.Wifi, tt.wantWifi) {
				t.Errorf("got wifi %v; want %v", s.Wifi, tt.wantWifi)
			}
			if !reflect.DeepEqual(s.AP, tt.wantAP) {
				t.Errorf("got ap %v; want %v", s.AP, tt.wantAP)
			}
		})
	}
}

func TestCaptureUnknownSection(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	if _, err := Capture(context.Background(), c, []string{"bogus"}, false); err == nil {
		t.Fatal("expected error for unknown section, got nil")
	}
}

func TestApply(t *testing.T) {
	posted := map[string][]byte{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		posted[r.URL.Path[1:]] = body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	s := &Snapshot{
		Display: map[string]any{"brightness": 150},
		Tickers: []tickers.Entry{
			{Type: tickers.TypeStocks, Ticker: "aapl", Time: tickers.Time15Min, Currency: tickers.CurrencyEUR},
		},
	}

	if err := Apply(context.Background(), c, s, []string{SectionDisplay, SectionTickers}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	var gotDisplay map[string]any
	if err := json.Unmarshal(posted[pathSettingsState], &gotDisplay); err != nil {
		t.Fatalf("decode posted display: %v", err)
	}
	if gotDisplay["brightness"] != float64(150) {
		t.Errorf("posted display brightness = %v; want 150", gotDisplay["brightness"])
	}

	var gotTickers tickers.State
	if err := json.Unmarshal(posted[pathCoinSetupState], &gotTickers); err != nil {
		t.Fatalf("decode posted tickers: %v", err)
	}
	want := tickers.State{Size: 1, Types: tickers.TypeStocks, Tickers: "AAPL", Times: tickers.Time15Min, Currency: tickers.CurrencyUSD}
	if gotTickers != want {
		t.Errorf("posted coinSetupState = %+v; want %+v (non-crypto forced to USD)", gotTickers, want)
	}
}

func TestApplyRejectsBlankTicker(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	s := &Snapshot{Tickers: []tickers.Entry{{Type: "", Ticker: "  ", Time: "", Currency: ""}}}

	if err := Apply(context.Background(), c, s, []string{SectionTickers}); err == nil {
		t.Fatal("Apply() with a blank ticker = nil error; want error")
	}
}

func TestApplyRejectsInvalidDisplayField(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	s := &Snapshot{Display: map[string]any{"brightness": float64(9999)}}

	if err := Apply(context.Background(), c, s, []string{SectionDisplay}); err == nil {
		t.Fatal("Apply() with brightness 9999 = nil error; want error")
	}
}

func TestApplyRejectsInvalidAPField(t *testing.T) {
	tests := []struct {
		name string
		ap   map[string]any
	}{
		{name: "channel out of range", ap: map[string]any{"channel": float64(99)}},
		{name: "malformed local_ip", ap: map[string]any{"local_ip": "not-an-ip"}},
		{name: "malformed subnet_mask", ap: map[string]any{"subnet_mask": "255.0.255.0"}},
		{name: "oversized ssid", ap: map[string]any{"ssid": strings.Repeat("a", 33)}},
		{name: "short password", ap: map[string]any{"password": "short"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := newTestServer(t, nil)
			defer srv.Close()
			c := client.New(srv.URL+"/", 0)
			s := &Snapshot{AP: tt.ap}
			if err := Apply(context.Background(), c, s, []string{SectionAP}); err == nil {
				t.Fatalf("Apply() with AP %v = nil error; want error", tt.ap)
			}
		})
	}
}

func TestApplyAcceptsValidAPField(t *testing.T) {
	posted := map[string][]byte{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		posted[r.URL.Path[1:]] = body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	s := &Snapshot{AP: map[string]any{
		"ssid": "box-ap", "channel": float64(6), "max_clients": float64(4),
		"local_ip": "192.168.4.1", "gateway_ip": "192.168.4.1", "subnet_mask": "255.255.255.0",
	}}
	if err := Apply(context.Background(), c, s, []string{SectionAP}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, ok := posted[pathAPSettings]; !ok {
		t.Fatal("Apply() never posted apSettings")
	}
}

func TestApplyRejectsInvalidTickerEnum(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	s := &Snapshot{Tickers: []tickers.Entry{{Type: "crytpo", Ticker: "BTC", Time: "5min", Currency: "USD"}}}

	if err := Apply(context.Background(), c, s, []string{SectionTickers}); err == nil {
		t.Fatal("Apply() with an invalid type = nil error; want error")
	}
}

func TestApplyUnknownSection(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	if err := Apply(context.Background(), c, &Snapshot{}, []string{"bogus"}); err == nil {
		t.Fatal("expected error for unknown section, got nil")
	}
}

func TestCaptureTickersDecodeError(t *testing.T) {
	bodies := map[string]string{
		pathCoinSetupState: `{"size":-1}`,
	}
	srv, _ := newTestServer(t, bodies)
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	if _, err := Capture(context.Background(), c, []string{SectionTickers}, false); err == nil {
		t.Fatal("expected error from a malformed coinSetupState response, got nil")
	}
}
