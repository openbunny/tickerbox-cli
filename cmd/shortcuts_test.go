// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/client"
)

func TestClampBrightness(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"below min", 0, minBrightness},
		{"at min", minBrightness, minBrightness},
		{"mid range", 128, 128},
		{"at max", maxBrightness, maxBrightness},
		{"above max", 999, maxBrightness},
		{"far below min", -50, minBrightness},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampBrightness(tt.in); got != tt.want {
				t.Errorf("clampBrightness(%d) = %d; want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseBrightnessValue(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{name: "min boundary", in: "10", want: 10},
		{name: "max boundary", in: "255", want: 255},
		{name: "mid value", in: "128", want: 128},
		{name: "below min", in: "9", wantErr: true},
		{name: "above max", in: "256", wantErr: true},
		{name: "not a number", in: "bright", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBrightnessValue(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("parseBrightnessValue(%q) = %d; want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseBrightnessStep(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    int
		wantErr bool
	}{
		{name: "no args uses default", args: nil, want: defaultBrightnessStep},
		{name: "explicit step", args: []string{"5"}, want: 5},
		{name: "zero step invalid", args: []string{"0"}, wantErr: true},
		{name: "negative step invalid", args: []string{"-5"}, wantErr: true},
		{name: "not a number", args: []string{"lots"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBrightnessStep(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %v, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("parseBrightnessStep(%v) = %d; want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{"zero", 0, "0s"},
		{"seconds only", 45, "45s"},
		{"minutes and seconds", 90, "1m 30s"},
		{"hours minutes seconds", 3661, "1h 1m 1s"},
		{"exactly one day", 86400, "1d 0h 0m 0s"},
		{"days hours minutes seconds", 90061, "1d 1h 1m 1s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatUptime(tt.seconds); got != tt.want {
				t.Errorf("formatUptime(%d) = %q; want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func brightnessServer(t *testing.T, initial int, posted *displaySettings) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(displaySettings{
				Brightness:     initial,
				ChangeInterval: 30,
				SleepEnabled:   true,
				SleepStart:     "22:00",
				SleepEnd:       "07:00",
			})
		case http.MethodPost:
			if err := json.NewDecoder(r.Body).Decode(posted); err != nil {
				t.Fatalf("decode posted body: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
}

func TestBrightnessMutations(t *testing.T) {
	tests := []struct {
		name    string
		initial int
		run     func(c *client.Client) (int, error)
		want    int
	}{
		{
			name:    "set within range",
			initial: 100,
			run:     func(c *client.Client) (int, error) { return setBrightness(context.Background(), c, 200) },
			want:    200,
		},
		{
			name:    "raise within range",
			initial: 100,
			run:     func(c *client.Client) (int, error) { return raiseBrightness(context.Background(), c, 30) },
			want:    130,
		},
		{
			name:    "raise clamps at max",
			initial: 250,
			run:     func(c *client.Client) (int, error) { return raiseBrightness(context.Background(), c, 20) },
			want:    maxBrightness,
		},
		{
			name:    "lower within range",
			initial: 100,
			run:     func(c *client.Client) (int, error) { return lowerBrightness(context.Background(), c, 30) },
			want:    70,
		},
		{
			name:    "lower clamps at min",
			initial: 15,
			run:     func(c *client.Client) (int, error) { return lowerBrightness(context.Background(), c, 20) },
			want:    minBrightness,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var posted displaySettings
			srv := brightnessServer(t, tt.initial, &posted)
			defer srv.Close()

			c := client.New(srv.URL+"/", 0)
			got, err := tt.run(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got brightness %d; want %d", got, tt.want)
			}
			if posted.Brightness != tt.want {
				t.Errorf("posted brightness %d; want %d", posted.Brightness, tt.want)
			}
			if posted.ChangeInterval != 30 || !posted.SleepEnabled || posted.SleepStart != "22:00" || posted.SleepEnd != "07:00" {
				t.Errorf("brightness mutation clobbered unrelated fields: %+v", posted)
			}
		})
	}
}

func TestMutateBrightnessPropagatesErrors(t *testing.T) {
	tests := []struct {
		name       string
		getStatus  int
		postStatus int
	}{
		{name: "get failure", getStatus: http.StatusInternalServerError, postStatus: http.StatusOK},
		{name: "post failure", getStatus: http.StatusOK, postStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					if tt.getStatus != http.StatusOK {
						w.WriteHeader(tt.getStatus)
						return
					}
					_ = json.NewEncoder(w).Encode(displaySettings{Brightness: 100})
					return
				}
				w.WriteHeader(tt.postStatus)
			}))
			defer srv.Close()

			c := client.New(srv.URL+"/", 0)
			if _, err := setBrightness(context.Background(), c, 150); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestDeviceUptime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ntpStatusPayload{Uptime: 125})
	}))
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	got, err := deviceUptime(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "2m 5s"; got != want {
		t.Errorf("deviceUptime() = %q; want %q", got, want)
	}
}

func TestDeviceUptimeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	if _, err := deviceUptime(context.Background(), c); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRebootDevice(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{name: "success", status: http.StatusOK},
		{name: "server error", status: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotLen int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotLen = r.ContentLength
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			c := client.New(srv.URL+"/", 0)
			err := rebootDevice(context.Background(), c)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPath != "/restart" {
				t.Errorf("got path %q; want %q", gotPath, "/restart")
			}
			if gotLen > 0 {
				t.Errorf("got content length %d; want empty body", gotLen)
			}
		})
	}
}

func TestSetDeviceTimezoneUnknownLabel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request for unknown label: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	if _, err := setDeviceTimezone(context.Background(), c, "Not/AReal"); err == nil {
		t.Fatal("expected error for unknown timezone label, got nil")
	}
}

func TestSetDeviceTimezoneWritesBothSections(t *testing.T) {
	const label = "Africa/Algiers"
	const wantPosix = "CET-1"

	var postedNTP ntpSettingsPayload
	var postedClock clockSettings

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/ntpSettings" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(ntpSettingsPayload{Enabled: true, Server: "pool.ntp.org", TZLabel: "Old/Label", TZFormat: "OLD0"})
		case r.URL.Path == "/ntpSettings" && r.Method == http.MethodPost:
			if err := json.NewDecoder(r.Body).Decode(&postedNTP); err != nil {
				t.Fatalf("decode posted ntpSettings: %v", err)
			}
		case r.URL.Path == "/clockSetupState" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(clockSettings{Enabled: true, TwelveHourFormat: true, AnimationSpeed: 50, TZLabel: "Old/Label"})
		case r.URL.Path == "/clockSetupState" && r.Method == http.MethodPost:
			if err := json.NewDecoder(r.Body).Decode(&postedClock); err != nil {
				t.Fatalf("decode posted clockSetupState: %v", err)
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL+"/", 0)
	posix, err := setDeviceTimezone(context.Background(), c, label)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if posix != wantPosix {
		t.Errorf("got posix %q; want %q", posix, wantPosix)
	}

	if postedNTP.TZLabel != label || postedNTP.TZFormat != wantPosix {
		t.Errorf("ntpSettings not updated: %+v", postedNTP)
	}
	if postedNTP.Enabled != true || postedNTP.Server != "pool.ntp.org" {
		t.Errorf("ntpSettings unrelated fields clobbered: %+v", postedNTP)
	}

	if postedClock.TZLabel != label {
		t.Errorf("clockSetupState not updated: %+v", postedClock)
	}
	if !postedClock.Enabled || !postedClock.TwelveHourFormat || postedClock.AnimationSpeed != 50 {
		t.Errorf("clockSetupState unrelated fields clobbered: %+v", postedClock)
	}

	if postedNTP.TZLabel != postedClock.TZLabel {
		t.Errorf("tz_label drifted between sections: ntp=%q clock=%q", postedNTP.TZLabel, postedClock.TZLabel)
	}
}

func TestSetDeviceTimezonePropagatesErrors(t *testing.T) {
	tests := []struct {
		name          string
		ntpGetStatus  int
		ntpPostStatus int
	}{
		{name: "ntp get failure", ntpGetStatus: http.StatusInternalServerError, ntpPostStatus: http.StatusOK},
		{name: "ntp post failure", ntpGetStatus: http.StatusOK, ntpPostStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/ntpSettings" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if r.Method == http.MethodGet {
					if tt.ntpGetStatus != http.StatusOK {
						w.WriteHeader(tt.ntpGetStatus)
						return
					}
					_ = json.NewEncoder(w).Encode(ntpSettingsPayload{})
					return
				}
				w.WriteHeader(tt.ntpPostStatus)
			}))
			defer srv.Close()

			c := client.New(srv.URL+"/", 0)
			if _, err := setDeviceTimezone(context.Background(), c, "Africa/Algiers"); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
