// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/openbunny/tickerbox-cli/internal/diff"
	"github.com/openbunny/tickerbox-cli/internal/profile"
	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func withCmdTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
}

func TestResolveInclude(t *testing.T) {
	tests := []struct {
		name        string
		includeFlag string
		all         bool
		wantInclude []string
		wantSecrets bool
	}{
		{
			name:        "no flags uses the default include set",
			includeFlag: "",
			all:         false,
			wantInclude: section.DefaultInclude,
			wantSecrets: false,
		},
		{
			name:        "explicit include, no all: secrets stay stripped",
			includeFlag: "tickers,display",
			all:         false,
			wantInclude: []string{"tickers", "display"},
			wantSecrets: false,
		},
		{
			name:        "include list trims whitespace and drops empty tokens",
			includeFlag: " tickers ,, display ,",
			all:         false,
			wantInclude: []string{"tickers", "display"},
			wantSecrets: false,
		},
		{
			name:        "all with no include adds wifi and ap to the default set",
			includeFlag: "",
			all:         true,
			wantInclude: []string{section.SectionTickers, section.SectionDisplay, section.SectionClock, section.SectionNTP, section.SectionWifi, section.SectionAP},
			wantSecrets: true,
		},
		{
			name:        "all with an explicit include appends wifi/ap once",
			includeFlag: "display",
			all:         true,
			wantInclude: []string{"display", section.SectionWifi, section.SectionAP},
			wantSecrets: true,
		},
		{
			name:        "all does not duplicate a wifi/ap already named in include",
			includeFlag: "wifi,display",
			all:         true,
			wantInclude: []string{"wifi", "display", section.SectionAP},
			wantSecrets: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			include, secrets := resolveInclude(tt.includeFlag, tt.all)
			if !reflect.DeepEqual(include, tt.wantInclude) {
				t.Errorf("include = %v; want %v", include, tt.wantInclude)
			}
			if secrets != tt.wantSecrets {
				t.Errorf("withSecrets = %v; want %v", secrets, tt.wantSecrets)
			}
		})
	}
}

func TestPresentSections(t *testing.T) {
	tests := []struct {
		name string
		s    *section.Snapshot
		want []string
	}{
		{
			name: "empty snapshot has no present sections",
			s:    &section.Snapshot{},
			want: nil,
		},
		{
			name: "only captured sections are present, in canonical order",
			s: &section.Snapshot{
				NTP:     map[string]any{"server": "pool.ntp.org"},
				Display: map[string]any{"brightness": float64(200)},
				Tickers: []tickers.Entry{{Type: tickers.TypeStocks, Ticker: "AAPL", Time: tickers.Time15Min, Currency: tickers.CurrencyUSD}},
			},
			want: []string{section.SectionTickers, section.SectionDisplay, section.SectionNTP},
		},
		{
			name: "an empty-but-non-nil map still counts as present",
			s:    &section.Snapshot{Wifi: map[string]any{}},
			want: []string{section.SectionWifi},
		},
		{
			name: "a zero-length ticker slice does not count as present",
			s:    &section.Snapshot{Tickers: []tickers.Entry{}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := presentSections(tt.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("presentSections() = %v; want %v", got, tt.want)
			}
		})
	}
}

func newDeviceServer(t *testing.T, bodies map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/rest/"):]
		body, ok := bodies[path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSaveThenDiffDetectsDeviceDrift(t *testing.T) {
	withCmdTempHome(t)

	bodies := map[string]string{
		"coinSetupState":  `{"size":1,"types":"stocks","tickers":"AAPL","times":"15min","currency":"USD"}`,
		"settingsState":   `{"brightness":200}`,
		"clockSetupState": `{"enabled":true}`,
		"ntpSettings":     `{"server":"pool.ntp.org"}`,
	}
	srv := newDeviceServer(t, bodies)

	origHost, origTimeout, origRetry := resolvedHost, timeoutFlag, retryFlag
	t.Cleanup(func() { resolvedHost, timeoutFlag, retryFlag = origHost, origTimeout, origRetry })
	resolvedHost, timeoutFlag, retryFlag = srv.URL, time.Second, 0

	include, withSecrets := resolveInclude("", false)
	captured, err := section.Capture(context.Background(), newClient(), include, withSecrets)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if err := profile.SaveDescribed("office", captured, "", ""); err != nil {
		t.Fatalf("SaveDescribed: %v", err)
	}

	prof, err := profile.Load("office")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	bodies["settingsState"] = `{"brightness":150}`

	deviceNow, err := section.Capture(context.Background(), newClient(), presentSections(prof), true)
	if err != nil {
		t.Fatalf("Capture after drift: %v", err)
	}

	diffs := diff.Diff(deviceNow, prof, presentSections(prof), false)
	want := []diff.FieldDiff{{Section: section.SectionDisplay, Field: "brightness", Device: float64(150), Saved: float64(200)}}
	if !reflect.DeepEqual(diffs, want) {
		t.Errorf("diffs = %+v; want %+v", diffs, want)
	}

	noDrift := diff.Diff(captured, prof, presentSections(prof), false)
	if len(noDrift) != 0 {
		t.Errorf("diff against the just-saved state = %+v; want none", noDrift)
	}
}

func TestProfileSaveRecordsSourceDevice(t *testing.T) {
	withCmdTempHome(t)

	bodies := map[string]string{
		"coinSetupState":  `{"size":0,"types":"","tickers":"","times":"","currency":""}`,
		"settingsState":   `{"brightness":200}`,
		"clockSetupState": `{"enabled":true}`,
		"ntpSettings":     `{"server":"pool.ntp.org"}`,
	}
	srv := newDeviceServer(t, bodies)

	origHost, origTimeout, origRetry := resolvedHost, timeoutFlag, retryFlag
	origDevice, origInclude, origAll, origDesc := resolvedDeviceName, profileSaveInclude, profileSaveAll, profileSaveDescription
	t.Cleanup(func() {
		resolvedHost, timeoutFlag, retryFlag = origHost, origTimeout, origRetry
		resolvedDeviceName, profileSaveInclude, profileSaveAll, profileSaveDescription = origDevice, origInclude, origAll, origDesc
	})
	resolvedHost, timeoutFlag, retryFlag = srv.URL, time.Second, 0
	profileSaveInclude, profileSaveAll, profileSaveDescription = "", false, ""

	tests := []struct {
		name        string
		deviceName  string
		profileName string
		wantSource  string
	}{
		{name: "records the resolved device name", deviceName: "kitchen", profileName: "src-named", wantSource: "kitchen"},
		{name: "omitted when saved via --host with no named device", deviceName: "", profileName: "src-hostonly", wantSource: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolvedDeviceName = tt.deviceName
			captureStdout(t, func() {
				if err := profileSaveCmd.RunE(profileSaveCmd, []string{tt.profileName}); err != nil {
					t.Fatalf("profileSaveCmd.RunE() = %v", err)
				}
			})

			_, gotSource, err := profile.LoadWithSource(tt.profileName)
			if err != nil {
				t.Fatalf("LoadWithSource(%q) = %v", tt.profileName, err)
			}
			if gotSource != tt.wantSource {
				t.Errorf("source_device = %q; want %q", gotSource, tt.wantSource)
			}

			dir, err := profile.Dir()
			if err != nil {
				t.Fatalf("profile.Dir() = %v", err)
			}
			raw, err := os.ReadFile(filepath.Join(dir, tt.profileName+".json"))
			if err != nil {
				t.Fatalf("read profile file: %v", err)
			}
			gotKey := strings.Contains(string(raw), "source_device")
			wantKey := tt.wantSource != ""
			if gotKey != wantKey {
				t.Errorf("profile file contains %q key = %v; want %v", "source_device", gotKey, wantKey)
			}
		})
	}
}

func TestProfileApplyConfirmsOnSourceMismatch(t *testing.T) {
	withCmdTempHome(t)

	if err := profile.SaveDescribed("mismatch", &section.Snapshot{Display: map[string]any{"brightness": float64(200)}}, "", "kitchen"); err != nil {
		t.Fatalf("SaveDescribed: %v", err)
	}

	origHost, origTimeout, origRetry := resolvedHost, timeoutFlag, retryFlag
	origDevice, origYes, origStdin := resolvedDeviceName, profileApplyYes, confirmStdin
	t.Cleanup(func() {
		resolvedHost, timeoutFlag, retryFlag = origHost, origTimeout, origRetry
		resolvedDeviceName, profileApplyYes, confirmStdin = origDevice, origYes, origStdin
	})
	timeoutFlag, retryFlag = time.Second, 0

	t.Run("decline exits 0 without modifying the device", func(t *testing.T) {
		var posted bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			posted = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}))
		t.Cleanup(srv.Close)
		resolvedHost, resolvedDeviceName, profileApplyYes = srv.URL, "office", false
		confirmStdin = strings.NewReader("n\n")

		var runErr error
		stdout := captureStdout(t, func() {
			runErr = profileApplyCmd.RunE(profileApplyCmd, []string{"mismatch"})
		})
		if runErr != nil {
			t.Fatalf("profileApplyCmd.RunE() = %v", runErr)
		}
		if !strings.Contains(stdout, "aborted") {
			t.Errorf("stdout = %q; want it to contain %q", stdout, "aborted")
		}
		if posted {
			t.Error("device was modified despite a declined confirmation")
		}
	})

	t.Run("confirm proceeds", func(t *testing.T) {
		var posted map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&posted)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}))
		t.Cleanup(srv.Close)
		resolvedHost, resolvedDeviceName, profileApplyYes = srv.URL, "office", false
		confirmStdin = strings.NewReader("y\n")

		if err := profileApplyCmd.RunE(profileApplyCmd, []string{"mismatch"}); err != nil {
			t.Fatalf("profileApplyCmd.RunE() = %v", err)
		}
		if posted["brightness"] != float64(200) {
			t.Errorf("posted brightness = %v; want 200", posted["brightness"])
		}
	})

	t.Run("--yes warns and skips the prompt", func(t *testing.T) {
		var posted map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&posted)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}))
		t.Cleanup(srv.Close)
		resolvedHost, resolvedDeviceName, profileApplyYes = srv.URL, "office", true

		var runErr error
		stderr := captureStderr(t, func() {
			runErr = profileApplyCmd.RunE(profileApplyCmd, []string{"mismatch"})
		})
		if runErr != nil {
			t.Fatalf("profileApplyCmd.RunE() = %v", runErr)
		}
		wantWarning := `warning: applying profile "mismatch" (saved from device "kitchen") to "office"`
		if !strings.Contains(stderr, wantWarning) {
			t.Errorf("stderr = %q; want it to contain %q", stderr, wantWarning)
		}
		if posted["brightness"] != float64(200) {
			t.Errorf("posted brightness = %v; want 200", posted["brightness"])
		}
	})
}

func TestProfileShowDisplaysSourceDevice(t *testing.T) {
	withCmdTempHome(t)

	if err := profile.SaveDescribed("shown", &section.Snapshot{
		Wifi: map[string]any{"ssid": "home", "password": "hunter2"},
	}, "notes", "kitchen"); err != nil {
		t.Fatalf("SaveDescribed: %v", err)
	}

	origJSON, origShowSecrets := jsonFlag, profileShowShowSecrets
	t.Cleanup(func() { jsonFlag, profileShowShowSecrets = origJSON, origShowSecrets })

	t.Run("table output masks the password and shows the source device", func(t *testing.T) {
		jsonFlag, profileShowShowSecrets = false, false
		var runErr error
		stdout := captureStdout(t, func() {
			runErr = profileShowCmd.RunE(profileShowCmd, []string{"shown"})
		})
		if runErr != nil {
			t.Fatalf("profileShowCmd.RunE() = %v", runErr)
		}
		if !strings.Contains(stdout, "source_device:") || !strings.Contains(stdout, "kitchen") {
			t.Errorf("stdout = %q; want it to show source_device kitchen", stdout)
		}
		if strings.Contains(stdout, "hunter2") {
			t.Errorf("stdout = %q; want the wifi password masked", stdout)
		}
		if !strings.Contains(stdout, "********") {
			t.Errorf("stdout = %q; want the masked placeholder", stdout)
		}
	})

	t.Run("json output includes source_device and masks the password unless --show-secrets", func(t *testing.T) {
		jsonFlag, profileShowShowSecrets = true, false
		var runErr error
		stdout := captureStdout(t, func() {
			runErr = profileShowCmd.RunE(profileShowCmd, []string{"shown"})
		})
		if runErr != nil {
			t.Fatalf("profileShowCmd.RunE() = %v", runErr)
		}
		var got struct {
			SourceDevice string         `json:"source_device"`
			Wifi         map[string]any `json:"wifi"`
		}
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("unmarshal %q: %v", stdout, err)
		}
		if got.SourceDevice != "kitchen" {
			t.Errorf("source_device = %q; want %q", got.SourceDevice, "kitchen")
		}
		if got.Wifi["password"] != "********" {
			t.Errorf("wifi.password = %v; want masked", got.Wifi["password"])
		}
	})

	t.Run("--show-secrets reveals the real password", func(t *testing.T) {
		jsonFlag, profileShowShowSecrets = true, true
		var runErr error
		stdout := captureStdout(t, func() {
			runErr = profileShowCmd.RunE(profileShowCmd, []string{"shown"})
		})
		if runErr != nil {
			t.Fatalf("profileShowCmd.RunE() = %v", runErr)
		}
		if !strings.Contains(stdout, "hunter2") {
			t.Errorf("stdout = %q; want the real password with --show-secrets", stdout)
		}
	})
}
