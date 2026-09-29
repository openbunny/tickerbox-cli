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

func TestDisplaySetRejectsConflictingSleepFlags(t *testing.T) {
	c := displaySetCmdFixture()
	mustSet(t, c, "sleep", "true")
	mustSet(t, c, "no-sleep", "true")

	if err := c.RunE(c, nil); err == nil {
		t.Fatal("expected error for mutually exclusive --sleep/--no-sleep")
	}
}

func TestDisplaySetRejectsBrightnessOutOfRange(t *testing.T) {
	c := displaySetCmdFixture()
	mustSet(t, c, "brightness", "5")

	if err := c.RunE(c, nil); err == nil {
		t.Fatal("expected error for brightness below 10")
	}
}

func TestDisplaySetRejectsInvalidSleepStart(t *testing.T) {
	c := displaySetCmdFixture()
	mustSet(t, c, "sleep-start", "not-a-time")

	if err := c.RunE(c, nil); err == nil {
		t.Fatal("expected error for an invalid --sleep-start")
	}
}

func TestClockSetRejectsConflictingEnabledFlags(t *testing.T) {
	c := clockSetCmdFixture()
	mustSet(t, c, "enabled", "true")
	mustSet(t, c, "disabled", "true")

	if err := c.RunE(c, nil); err == nil {
		t.Fatal("expected error for mutually exclusive --enabled/--disabled")
	}
}

func TestClockSetRejectsConflictingHourFormatFlags(t *testing.T) {
	c := clockSetCmdFixture()
	mustSet(t, c, "12h", "true")
	mustSet(t, c, "24h", "true")

	if err := c.RunE(c, nil); err == nil {
		t.Fatal("expected error for mutually exclusive --12h/--24h")
	}
}

func TestClockSetRejectsUnknownTimezone(t *testing.T) {
	c := clockSetCmdFixture()
	mustSet(t, c, "tz", "Not/A_Real_Zone")

	if err := c.RunE(c, nil); err == nil {
		t.Fatal("expected error for an unknown --tz label")
	}
}

func TestClockSetRejectsAnimationSpeedOutOfRange(t *testing.T) {
	c := clockSetCmdFixture()
	mustSet(t, c, "animation-speed", "5")

	if err := c.RunE(c, nil); err == nil {
		t.Fatal("expected error for animation-speed below 10")
	}
}

func mustSet(t *testing.T, c *cobra.Command, name, value string) {
	t.Helper()
	if err := c.Flags().Set(name, value); err != nil {
		t.Fatalf("set --%s=%s: %v", name, value, err)
	}
}

func TestDisplaySettingsCmdRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(displaySettings{Brightness: 200, ChangeInterval: 20})
	}))
	t.Cleanup(srv.Close)
	withCmdTarget(t, srv.URL)

	if err := displaySettingsCmd.RunE(displaySettingsCmd, nil); err != nil {
		t.Fatalf("displaySettingsCmd.RunE() = %v", err)
	}
}

func TestClockSettingsCmdRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(clockSettings{Enabled: true, TZLabel: "UTC"})
	}))
	t.Cleanup(srv.Close)
	withCmdTarget(t, srv.URL)

	if err := clockSettingsCmd.RunE(clockSettingsCmd, nil); err != nil {
		t.Fatalf("clockSettingsCmd.RunE() = %v", err)
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
