// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

func TestValidateISO8601(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"RFC3339", "2024-01-01T10:00:00Z", false},
		{"device layout", "2024-01-01T10:00:00", false},
		{"date only", "2024-01-01", false},
		{"garbage", "not-a-date", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateISO8601(tt.in)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q, got nil", tt.in)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.in, err)
			}
		})
	}
}

func TestYesNo(t *testing.T) {
	if got := yesNo(true); got != "yes" {
		t.Errorf("yesNo(true) = %q; want yes", got)
	}
	if got := yesNo(false); got != "no" {
		t.Errorf("yesNo(false) = %q; want no", got)
	}
}

func displaySetCmdFixture() *cobra.Command {
	c := &cobra.Command{Use: "set", RunE: displaySetCmd.RunE}
	c.Flags().Int("brightness", 0, "")
	c.Flags().Int("interval", 0, "")
	c.Flags().Bool("sleep", false, "")
	c.Flags().Bool("no-sleep", false, "")
	c.Flags().String("sleep-start", "", "")
	c.Flags().String("sleep-end", "", "")
	return c
}

func clockSetCmdFixture() *cobra.Command {
	c := &cobra.Command{Use: "set", RunE: clockSetCmd.RunE}
	c.Flags().Bool("enabled", false, "")
	c.Flags().Bool("disabled", false, "")
	c.Flags().Bool("12h", false, "")
	c.Flags().Bool("24h", false, "")
	c.Flags().Int("animation-speed", 0, "")
	c.Flags().String("tz", "", "")
	return c
}

func TestDisplaySetRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]string
	}{
		{"conflicting sleep flags", map[string]string{"sleep": "true", "no-sleep": "true"}},
		{"brightness below 10", map[string]string{"brightness": "5"}},
		{"invalid sleep-start", map[string]string{"sleep-start": "not-a-time"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := displaySetCmdFixture()
			for name, value := range tt.flags {
				mustSet(t, c, name, value)
			}
			if err := c.RunE(c, nil); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestClockSetRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]string
	}{
		{"conflicting enabled flags", map[string]string{"enabled": "true", "disabled": "true"}},
		{"conflicting hour format flags", map[string]string{"12h": "true", "24h": "true"}},
		{"unknown timezone", map[string]string{"tz": "Not/A_Real_Zone"}},
		{"animation-speed below 10", map[string]string{"animation-speed": "5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clockSetCmdFixture()
			for name, value := range tt.flags {
				mustSet(t, c, name, value)
			}
			if err := c.RunE(c, nil); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func mustSet(t *testing.T, c *cobra.Command, name, value string) {
	t.Helper()
	if err := c.Flags().Set(name, value); err != nil {
		t.Fatalf("set --%s=%s: %v", name, value, err)
	}
}

func TestSettingsCmdRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		cmd     *cobra.Command
	}{
		{"display settings", displaySettings{Brightness: 200, ChangeInterval: 20}, displaySettingsCmd},
		{"clock settings", clockSettings{Enabled: true, TZLabel: "UTC"}, clockSettingsCmd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.payload)
			}))
			t.Cleanup(srv.Close)
			withCmdTarget(t, srv.URL)

			if err := tt.cmd.RunE(tt.cmd, nil); err != nil {
				t.Fatalf("%s.RunE() = %v", tt.cmd.Name(), err)
			}
		})
	}
}

func TestDisplaySetAppliesChangedFieldsOnly(t *testing.T) {
	var posted displaySettings
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(displaySettings{Brightness: 50, ChangeInterval: 15, SleepEnabled: false})
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&posted)
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)
	withCmdTarget(t, srv.URL)

	c := displaySetCmdFixture()
	mustSet(t, c, "brightness", "180")
	mustSet(t, c, "sleep", "true")

	if err := c.RunE(c, nil); err != nil {
		t.Fatalf("displaySetCmd.RunE() = %v", err)
	}
	if posted.Brightness != 180 {
		t.Errorf("posted brightness = %d; want 180", posted.Brightness)
	}
	if posted.ChangeInterval != 15 {
		t.Errorf("posted change interval = %d; want unchanged 15", posted.ChangeInterval)
	}
	if !posted.SleepEnabled {
		t.Error("posted sleep enabled = false; want true")
	}
}
