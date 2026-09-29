// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
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
	TwelweHourFormat bool   `json:"twelweHourFormat"`
	AnimationSpeed   int    `json:"animationSpeed"`
	TZLabel          string `json:"tz_label"`
}

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

func renderDisplaySettings(s displaySettings) {
	output.KV([][2]string{
		{"Brightness", fmt.Sprintf("%d", s.Brightness)},
		{"Change interval (s)", fmt.Sprintf("%d", s.ChangeInterval)},
		{"Sleep enabled", yesNo(s.SleepEnabled)},
		{"Sleep start", s.SleepStart},
		{"Sleep end", s.SleepEnd},
	})
}

func renderClockSettings(s clockSettings) {
	output.KV([][2]string{
		{"Enabled", yesNo(s.Enabled)},
		{"12-hour format", yesNo(s.TwelweHourFormat)},
		{"Animation speed", fmt.Sprintf("%d", s.AnimationSpeed)},
		{"Timezone", s.TZLabel},
	})
}

var displayCmd = &cobra.Command{
	Use:   "display",
	Short: "Display brightness / rotation / sleep",
}

var displaySettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show display settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		var state displaySettings
		if err := newClient().Get("settingsState", &state); err != nil {
			return fmt.Errorf("get display settings: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(state)
		}
		renderDisplaySettings(state)
		return nil
	},
}

var displaySetCmd = &cobra.Command{
	Use:   "set",
	Short: "Change display settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		brightness, _ := flags.GetInt("brightness")
		interval, _ := flags.GetInt("interval")
		sleep, _ := flags.GetBool("sleep")
		noSleep, _ := flags.GetBool("no-sleep")
		sleepStart, _ := flags.GetString("sleep-start")
		sleepEnd, _ := flags.GetString("sleep-end")

		if sleep && noSleep {
			return errors.New("--sleep and --no-sleep are mutually exclusive")
		}
		if flags.Changed("brightness") && (brightness < 10 || brightness > 255) {
			return fmt.Errorf("brightness must be between 10 and 255, got %d", brightness)
		}
		if flags.Changed("interval") && (interval < 10 || interval > 60) {
			return fmt.Errorf("interval must be between 10 and 60, got %d", interval)
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
		if err := c.Get("settingsState", &state); err != nil {
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

		if err := c.Post("settingsState", state); err != nil {
			return fmt.Errorf("post display settings: %w", err)
		}

		if jsonOut() {
			return output.EmitJSON(state)
		}
		renderDisplaySettings(state)
		return nil
	},
}

var clockCmd = &cobra.Command{
	Use:   "clock",
	Short: "Clock screen settings",
}

var clockSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show clock settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		var state clockSettings
		if err := newClient().Get("clockSetupState", &state); err != nil {
			return fmt.Errorf("get clock settings: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(state)
		}
		renderClockSettings(state)
		return nil
	},
}

var clockSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Change clock settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		enabled, _ := flags.GetBool("enabled")
		disabled, _ := flags.GetBool("disabled")
		twelveHour, _ := flags.GetBool("12h")
		twentyFourHour, _ := flags.GetBool("24h")
		animationSpeed, _ := flags.GetInt("animation-speed")
		tzLabel, _ := flags.GetString("tz")

		if enabled && disabled {
			return errors.New("--enabled and --disabled are mutually exclusive")
		}
		if twelveHour && twentyFourHour {
			return errors.New("--12h and --24h are mutually exclusive")
		}
		if flags.Changed("animation-speed") && (animationSpeed < 10 || animationSpeed > 200) {
			return fmt.Errorf("animation-speed must be between 10 and 200, got %d", animationSpeed)
		}
		if flags.Changed("tz") {
			if _, ok := tz.PosixFor(tzLabel); !ok {
				return fmt.Errorf("tz must be a known timezone label, got %q", tzLabel)
			}
		}

		c := newClient()
		var state clockSettings
		if err := c.Get("clockSetupState", &state); err != nil {
			return fmt.Errorf("get clock settings: %w", err)
		}

		if enabled {
			state.Enabled = true
		}
		if disabled {
			state.Enabled = false
		}
		if twelveHour {
			state.TwelweHourFormat = true
		}
		if twentyFourHour {
			state.TwelweHourFormat = false
		}
		if flags.Changed("animation-speed") {
			state.AnimationSpeed = animationSpeed
		}
		if flags.Changed("tz") {
			state.TZLabel = tzLabel
		}

		if err := c.Post("clockSetupState", state); err != nil {
			return fmt.Errorf("post clock settings: %w", err)
		}

		if jsonOut() {
			return output.EmitJSON(state)
		}
		renderClockSettings(state)
		return nil
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

	clockSetCmd.Flags().Bool("enabled", false, "enable the clock screen")
	clockSetCmd.Flags().Bool("disabled", false, "disable the clock screen")
	clockSetCmd.Flags().Bool("12h", false, "use 12-hour time format")
	clockSetCmd.Flags().Bool("24h", false, "use 24-hour time format")
	clockSetCmd.Flags().Int("animation-speed", 0, "clock animation speed (10-200)")
	clockSetCmd.Flags().String("tz", "", "timezone label")
}
