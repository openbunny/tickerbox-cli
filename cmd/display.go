// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/tz"
)

type displaySettings struct {
	Brightness     int    `json:"brightness"`
	ChangeInterval int    `json:"changeInterval"`
	SleepEnabled   bool   `json:"sleepEnabled"`
	SleepStart     string `json:"sleepStart"`
	SleepEnd       string `json:"sleepEnd"`
}

type clockSettings struct {
	Enabled          bool   `json:"enabled"`
	TwelveHourFormat bool   `json:"twelweHourFormat"`
	AnimationSpeed   int    `json:"animationSpeed"`
	TZLabel          string `json:"tz_label"`
}

const (
	minDisplayInterval = 10
	maxDisplayInterval = 60

	minAnimationSpeed = 10
	maxAnimationSpeed = 200
)

var iso8601Layouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func validateISO8601(value string) error {
	for _, layout := range iso8601Layouts {
		if _, err := time.Parse(layout, value); err == nil {
			return nil
		}
	}
	return fmt.Errorf("invalid ISO8601 value %q", value)
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func renderDisplaySettings(s displaySettings) error {
	return output.KV([][2]string{
		{"Brightness", fmt.Sprintf("%d", s.Brightness)},
		{"Change interval (s)", fmt.Sprintf("%d", s.ChangeInterval)},
		{"Sleep enabled", yesNo(s.SleepEnabled)},
		{"Sleep start", s.SleepStart},
		{"Sleep end", s.SleepEnd},
	})
}

func renderClockSettings(s clockSettings) error {
	return output.KV([][2]string{
		{"Enabled", yesNo(s.Enabled)},
		{"12-hour format", yesNo(s.TwelveHourFormat)},
		{"Animation speed", fmt.Sprintf("%d", s.AnimationSpeed)},
		{"Timezone", s.TZLabel},
	})
}

var displayCmd = &cobra.Command{
	Use:   "display",
	Short: "Display brightness / rotation / sleep",
}

var displaySettingsCmd = &cobra.Command{
	Use:     "settings",
	Short:   "Show display settings",
	Example: "  tickerbox display settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		var state displaySettings
		if err := newClient().Get(cmdContext(cmd), "settingsState", &state); err != nil {
			return fmt.Errorf("get display settings: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(state)
		}
		return renderDisplaySettings(state)
	},
}

var displaySetCmd = &cobra.Command{
	Use:   "set",
	Short: "Change display settings",
	Long: "Updates only the fields given as flags. --brightness is 10-255, --interval is 10-60 seconds " +
		"between screens. --sleep and --no-sleep are mutually exclusive. --sleep-start and --sleep-end accept " +
		"RFC3339, 2006-01-02T15:04:05, or 2006-01-02.",
	Example: "  tickerbox display set --brightness 180 --interval 20 --sleep --sleep-start 2026-01-01T22:00:00 --sleep-end 2026-01-02T07:00:00",
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		brightness, _ := flags.GetInt("brightness")
		interval, _ := flags.GetInt("interval")
		sleep, _ := flags.GetBool("sleep")
		noSleep, _ := flags.GetBool("no-sleep")
		sleepStart, _ := flags.GetString("sleep-start")
		sleepEnd, _ := flags.GetString("sleep-end")

		if flags.Changed("brightness") && (brightness < minBrightness || brightness > maxBrightness) {
			return fmt.Errorf("brightness must be between %d and %d, got %d", minBrightness, maxBrightness, brightness)
		}
		if flags.Changed("interval") && (interval < minDisplayInterval || interval > maxDisplayInterval) {
			return fmt.Errorf("interval must be between %d and %d, got %d", minDisplayInterval, maxDisplayInterval, interval)
		}
		if flags.Changed("sleep-start") {
			if err := validateISO8601(sleepStart); err != nil {
				return fmt.Errorf("sleep-start: %w", err)
			}
		}
		if flags.Changed("sleep-end") {
			if err := validateISO8601(sleepEnd); err != nil {
				return fmt.Errorf("sleep-end: %w", err)
			}
		}

		c := newClient()
		var state displaySettings
		if err := c.Get(cmdContext(cmd), "settingsState", &state); err != nil {
			return fmt.Errorf("get display settings: %w", err)
		}

		if flags.Changed("brightness") {
			state.Brightness = brightness
		}
		if flags.Changed("interval") {
			state.ChangeInterval = interval
		}
		if sleep {
			state.SleepEnabled = true
		}
		if noSleep {
			state.SleepEnabled = false
		}
		if flags.Changed("sleep-start") {
			state.SleepStart = sleepStart
		}
		if flags.Changed("sleep-end") {
			state.SleepEnd = sleepEnd
		}

		if err := c.Post(cmdContext(cmd), "settingsState", state); err != nil {
			return fmt.Errorf("post display settings: %w", err)
		}

		if jsonOut() {
			return output.EmitJSON(state)
		}
		return renderDisplaySettings(state)
	},
}

var clockCmd = &cobra.Command{
	Use:   "clock",
	Short: "Clock screen settings",
}

var clockSettingsCmd = &cobra.Command{
	Use:     "settings",
	Short:   "Show clock settings",
	Example: "  tickerbox clock settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		var state clockSettings
		if err := newClient().Get(cmdContext(cmd), "clockSetupState", &state); err != nil {
			return fmt.Errorf("get clock settings: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(state)
		}
		return renderClockSettings(state)
	},
}

var clockSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Change clock settings",
	Long: "Updates only the fields given as flags. --enabled/--disabled and --12h/--24h are each mutually " +
		"exclusive. --tz must be a label from `tickerbox tz list`.",
	Example: "  tickerbox clock set --enabled --12h --tz America/Chicago",
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		enabled, _ := flags.GetBool("enabled")
		disabled, _ := flags.GetBool("disabled")
		twelveHour, _ := flags.GetBool("12h")
		twentyFourHour, _ := flags.GetBool("24h")
		animationSpeed, _ := flags.GetInt("animation-speed")
		tzLabel, _ := flags.GetString("tz")

		if flags.Changed("animation-speed") && (animationSpeed < minAnimationSpeed || animationSpeed > maxAnimationSpeed) {
			return fmt.Errorf("animation-speed must be between %d and %d, got %d", minAnimationSpeed, maxAnimationSpeed, animationSpeed)
		}
		if flags.Changed("tz") {
			if _, ok := tz.PosixFor(tzLabel); !ok {
				return fmt.Errorf("tz must be a known timezone label, got %q", tzLabel)
			}
		}

		c := newClient()
		var state clockSettings
		if err := c.Get(cmdContext(cmd), "clockSetupState", &state); err != nil {
			return fmt.Errorf("get clock settings: %w", err)
		}

		if enabled {
			state.Enabled = true
		}
		if disabled {
			state.Enabled = false
		}
		if twelveHour {
			state.TwelveHourFormat = true
		}
		if twentyFourHour {
			state.TwelveHourFormat = false
		}
		if flags.Changed("animation-speed") {
			state.AnimationSpeed = animationSpeed
		}
		if flags.Changed("tz") {
			state.TZLabel = tzLabel
		}

		if err := c.Post(cmdContext(cmd), "clockSetupState", state); err != nil {
			return fmt.Errorf("post clock settings: %w", err)
		}

		if jsonOut() {
			return output.EmitJSON(state)
		}
		return renderClockSettings(state)
	},
}

func init() {
	rootCmd.AddCommand(displayCmd)
	rootCmd.AddCommand(clockCmd)

	displayCmd.AddCommand(displaySettingsCmd)
	displayCmd.AddCommand(displaySetCmd)

	clockCmd.AddCommand(clockSettingsCmd)
	clockCmd.AddCommand(clockSetCmd)

	displaySetCmd.Flags().Int("brightness", 0, "display brightness (10-255)")
	displaySetCmd.Flags().Int("interval", 0, "seconds between screens (10-60)")
	displaySetCmd.Flags().Bool("sleep", false, "enable scheduled sleep")
	displaySetCmd.Flags().Bool("no-sleep", false, "disable scheduled sleep")
	displaySetCmd.Flags().String("sleep-start", "", "sleep start time, ISO8601")
	displaySetCmd.Flags().String("sleep-end", "", "sleep end time, ISO8601")
	displaySetCmd.MarkFlagsMutuallyExclusive("sleep", "no-sleep")

	clockSetCmd.Flags().Bool("enabled", false, "enable the clock screen")
	clockSetCmd.Flags().Bool("disabled", false, "disable the clock screen")
	clockSetCmd.Flags().Bool("12h", false, "use 12-hour time format")
	clockSetCmd.Flags().Bool("24h", false, "use 24-hour time format")
	clockSetCmd.Flags().Int("animation-speed", 0, "clock animation speed (10-200)")
	clockSetCmd.Flags().String("tz", "", "timezone label")
	clockSetCmd.MarkFlagsMutuallyExclusive("enabled", "disabled")
	clockSetCmd.MarkFlagsMutuallyExclusive("12h", "24h")
}
