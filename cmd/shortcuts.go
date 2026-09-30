// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/tz"
)

const (
	minBrightness         = 10
	maxBrightness         = 255
	defaultBrightnessStep = 25
)

func clampBrightness(v int) int {
	switch {
	case v < minBrightness:
		return minBrightness
	case v > maxBrightness:
		return maxBrightness
	default:
		return v
	}
}

func parseBrightnessValue(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid brightness %q: %w", s, err)
	}
	if v < minBrightness || v > maxBrightness {
		return 0, fmt.Errorf("brightness must be between %d and %d, got %d", minBrightness, maxBrightness, v)
	}
	return v, nil
}

func parseBrightnessStep(args []string) (int, error) {
	if len(args) == 0 {
		return defaultBrightnessStep, nil
	}
	step, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, fmt.Errorf("invalid step %q: %w", args[0], err)
	}
	if step <= 0 {
		return 0, fmt.Errorf("step must be positive, got %d", step)
	}
	return step, nil
}

func mutateBrightness(ctx context.Context, c *client.Client, mutate func(int) int) (int, error) {
	var state displaySettings
	if err := c.Get(ctx, "settingsState", &state); err != nil {
		return 0, fmt.Errorf("get display settings: %w", err)
	}
	state.Brightness = mutate(state.Brightness)
	if err := c.Post(ctx, "settingsState", state); err != nil {
		return 0, fmt.Errorf("set brightness: %w", err)
	}
	return state.Brightness, nil
}

func setBrightness(ctx context.Context, c *client.Client, val int) (int, error) {
	return mutateBrightness(ctx, c, func(int) int { return val })
}

func raiseBrightness(ctx context.Context, c *client.Client, step int) (int, error) {
	return mutateBrightness(ctx, c, func(cur int) int { return clampBrightness(cur + step) })
}

func lowerBrightness(ctx context.Context, c *client.Client, step int) (int, error) {
	return mutateBrightness(ctx, c, func(cur int) int { return clampBrightness(cur - step) })
}

func renderBrightness(val int) error {
	if jsonOut() {
		return output.EmitJSON(map[string]int{"brightness": val})
	}
	return output.KV([][2]string{{"brightness", strconv.Itoa(val)}})
}

func formatUptime(seconds int) string {
	d := time.Duration(seconds) * time.Second
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	secs := d / time.Second

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if days > 0 || hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if days > 0 || hours > 0 || minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	parts = append(parts, fmt.Sprintf("%ds", secs))
	return strings.Join(parts, " ")
}

func deviceUptime(ctx context.Context, c *client.Client) (string, error) {
	var st ntpStatusPayload
	if err := c.Get(ctx, "ntpStatus", &st); err != nil {
		return "", fmt.Errorf("get uptime: %w", err)
	}
	return formatUptime(st.Uptime), nil
}

func rebootDevice(ctx context.Context, c *client.Client) error {
	if err := c.Post(ctx, "restart", nil); err != nil {
		return fmt.Errorf("restart device: %w", err)
	}
	return nil
}

func setNTPTimezone(ctx context.Context, c *client.Client, label, posix string) error {
	var settings ntpSettingsPayload
	if err := c.Get(ctx, "ntpSettings", &settings); err != nil {
		return fmt.Errorf("get ntp settings: %w", err)
	}
	settings.TZLabel = label
	settings.TZFormat = posix
	if err := c.Post(ctx, "ntpSettings", settings); err != nil {
		return fmt.Errorf("set ntp settings: %w", err)
	}
	return nil
}

func setClockTimezone(ctx context.Context, c *client.Client, label string) error {
	var state clockSettings
	if err := c.Get(ctx, "clockSetupState", &state); err != nil {
		return fmt.Errorf("get clock settings: %w", err)
	}
	state.TZLabel = label
	if err := c.Post(ctx, "clockSetupState", state); err != nil {
		return fmt.Errorf("set clock settings: %w", err)
	}
	return nil
}

func setDeviceTimezone(ctx context.Context, c *client.Client, label string) (string, error) {
	posix, ok := tz.PosixFor(label)
	if !ok {
		return "", fmt.Errorf("unknown timezone label %q", label)
	}
	if err := setNTPTimezone(ctx, c, label, posix); err != nil {
		return "", err
	}
	if err := setClockTimezone(ctx, c, label); err != nil {
		return "", err
	}
	return posix, nil
}

var rebootYes bool

var rebootCmd = &cobra.Command{
	Use:     "reboot",
	Short:   "Restart the device (alias for system restart)",
	Long:    "Alias for `system restart`. Prompts for confirmation unless --yes.",
	Example: "  tickerbox reboot --yes",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !rebootYes {
			ok, err := confirm("Restart the device?")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := rebootDevice(cmdContext(cmd), newClient()); err != nil {
			return err
		}
		fmt.Println("restart requested")
		return nil
	},
}

var brightnessCmd = &cobra.Command{
	Use:   "brightness [10-255]",
	Short: "Set display brightness",
	Long: "Sets brightness directly to a value from 10-255. Use the up/down subcommands to step relative " +
		"to the current value instead.",
	Example: "  tickerbox brightness 180",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("brightness requires a value between %d and %d, or a subcommand (up, down)", minBrightness, maxBrightness)
		}
		val, err := parseBrightnessValue(args[0])
		if err != nil {
			return err
		}
		got, err := setBrightness(cmdContext(cmd), newClient(), val)
		if err != nil {
			return err
		}
		return renderBrightness(got)
	},
}

var brightnessUpCmd = &cobra.Command{
	Use:     "up [step]",
	Short:   "Increase display brightness, clamped to 10-255",
	Long:    "Raises brightness by step, or by 25 if step is omitted, clamped to 10-255.",
	Example: "  tickerbox brightness up 15",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		step, err := parseBrightnessStep(args)
		if err != nil {
			return err
		}
		got, err := raiseBrightness(cmdContext(cmd), newClient(), step)
		if err != nil {
			return err
		}
		return renderBrightness(got)
	},
}

var brightnessDownCmd = &cobra.Command{
	Use:     "down [step]",
	Short:   "Decrease display brightness, clamped to 10-255",
	Long:    "Lowers brightness by step, or by 25 if step is omitted, clamped to 10-255.",
	Example: "  tickerbox brightness down",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		step, err := parseBrightnessStep(args)
		if err != nil {
			return err
		}
		got, err := lowerBrightness(cmdContext(cmd), newClient(), step)
		if err != nil {
			return err
		}
		return renderBrightness(got)
	},
}

var uptimeCmd = &cobra.Command{
	Use:     "uptime",
	Short:   "Show device uptime since last boot",
	Example: "  tickerbox uptime",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		formatted, err := deviceUptime(cmdContext(cmd), newClient())
		if err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(map[string]string{"uptime": formatted})
		}
		return output.KV([][2]string{{"uptime", formatted}})
	},
}

var tzSetCmd = &cobra.Command{
	Use:   "set <label>",
	Short: "Set the device timezone",
	Long: "label must be one shown by `tickerbox tz list`. Sets both the ntp and clock timezone fields on " +
		"the device.",
	Example: "  tickerbox tz set America/New_York",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label := args[0]
		posix, err := setDeviceTimezone(cmdContext(cmd), newClient(), label)
		if err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(map[string]string{"tz_label": label, "tz_format": posix})
		}
		return output.KV([][2]string{{"tz_label", label}, {"tz_format", posix}})
	},
}

func init() {
	rebootCmd.Flags().BoolVarP(&rebootYes, "yes", "y", false, "skip confirmation")

	brightnessCmd.AddCommand(brightnessUpCmd, brightnessDownCmd)

	tzCmd.AddCommand(tzSetCmd)

	rootCmd.AddCommand(rebootCmd, brightnessCmd, uptimeCmd)
}
