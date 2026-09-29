package cmd

import (
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

// mutateBrightness fetches settingsState, applies mutate to its Brightness
// field, and posts the result back, preserving every other field so a
// brightness change never clobbers change interval or sleep settings.
func mutateBrightness(c *client.Client, mutate func(int) int) (int, error) {
	var state displaySettings
	if err := c.Get("settingsState", &state); err != nil {
		return 0, fmt.Errorf("get display settings: %w", err)
	}
	state.Brightness = mutate(state.Brightness)
	if err := c.Post("settingsState", state); err != nil {
		return 0, fmt.Errorf("set brightness: %w", err)
	}
	return state.Brightness, nil
}

func setBrightness(c *client.Client, val int) (int, error) {
	return mutateBrightness(c, func(int) int { return val })
}

func raiseBrightness(c *client.Client, step int) (int, error) {
	return mutateBrightness(c, func(cur int) int { return clampBrightness(cur + step) })
}

func lowerBrightness(c *client.Client, step int) (int, error) {
	return mutateBrightness(c, func(cur int) int { return clampBrightness(cur - step) })
}

func renderBrightness(val int) error {
	if jsonOut() {
		return output.EmitJSON(map[string]int{"brightness": val})
	}
	output.KV([][2]string{{"brightness", strconv.Itoa(val)}})
	return nil
}

// formatUptime renders a device-reported uptime in seconds as a human
// duration, e.g. "1d 2h 3m 4s". Zero-valued leading units are omitted.
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

func deviceUptime(c *client.Client) (string, error) {
	var st ntpStatusPayload
	if err := c.Get("ntpStatus", &st); err != nil {
		return "", fmt.Errorf("get uptime: %w", err)
	}
	return formatUptime(st.Uptime), nil
}

func rebootDevice(c *client.Client) error {
	if err := c.Post("restart", nil); err != nil {
		return fmt.Errorf("restart device: %w", err)
	}
	return nil
}

func setNTPTimezone(c *client.Client, label, posix string) error {
	var settings ntpSettingsPayload
	if err := c.Get("ntpSettings", &settings); err != nil {
		return fmt.Errorf("get ntp settings: %w", err)
	}
	settings.TZLabel = label
	settings.TZFormat = posix
	if err := c.Post("ntpSettings", settings); err != nil {
		return fmt.Errorf("set ntp settings: %w", err)
	}
	return nil
}

func setClockTimezone(c *client.Client, label string) error {
	var state clockSettings
	if err := c.Get("clockSetupState", &state); err != nil {
		return fmt.Errorf("get clock settings: %w", err)
	}
	state.TZLabel = label
	if err := c.Post("clockSetupState", state); err != nil {
		return fmt.Errorf("set clock settings: %w", err)
	}
	return nil
}

// setDeviceTimezone validates label and writes it to both ntpSettings and
// clockSetupState, so the two never drift apart.
func setDeviceTimezone(c *client.Client, label string) (string, error) {
	posix, ok := tz.PosixFor(label)
	if !ok {
		return "", fmt.Errorf("unknown timezone label %q", label)
	}
	if err := setNTPTimezone(c, label, posix); err != nil {
		return "", err
	}
	if err := setClockTimezone(c, label); err != nil {
		return "", err
	}
	return posix, nil
}

var rebootYes bool

var rebootCmd = &cobra.Command{
	Use:   "reboot",
	Short: "Restart the device (alias for system restart)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !rebootYes {
			ok, err := confirmSystemAction("Restart the device?")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := rebootDevice(newClient()); err != nil {
			return err
		}
		fmt.Println("restart requested")
		return nil
	},
}

var brightnessCmd = &cobra.Command{
	Use:   "brightness [10-255]",
	Short: "Set display brightness",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("brightness requires a value between %d and %d, or a subcommand (up, down)", minBrightness, maxBrightness)
		}
		val, err := parseBrightnessValue(args[0])
		if err != nil {
			return err
		}
		got, err := setBrightness(newClient(), val)
		if err != nil {
			return err
		}
		return renderBrightness(got)
	},
}

var brightnessUpCmd = &cobra.Command{
	Use:   "up [step]",
	Short: "Increase display brightness, clamped to 10-255",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		step, err := parseBrightnessStep(args)
		if err != nil {
			return err
		}
		got, err := raiseBrightness(newClient(), step)
		if err != nil {
			return err
		}
		return renderBrightness(got)
	},
}

var brightnessDownCmd = &cobra.Command{
	Use:   "down [step]",
	Short: "Decrease display brightness, clamped to 10-255",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		step, err := parseBrightnessStep(args)
		if err != nil {
			return err
		}
		got, err := lowerBrightness(newClient(), step)
		if err != nil {
			return err
		}
		return renderBrightness(got)
	},
}

var uptimeCmd = &cobra.Command{
	Use:   "uptime",
	Short: "Show device uptime since last boot",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		formatted, err := deviceUptime(newClient())
		if err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(map[string]string{"uptime": formatted})
		}
		output.KV([][2]string{{"uptime", formatted}})
		return nil
	},
}

var tzSetCmd = &cobra.Command{
	Use:   "set <label>",
	Short: "Set the device timezone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label := args[0]
		posix, err := setDeviceTimezone(newClient(), label)
		if err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(map[string]string{"tz_label": label, "tz_format": posix})
		}
		output.KV([][2]string{{"tz_label", label}, {"tz_format", posix}})
		return nil
	},
}

func init() {
	rebootCmd.Flags().BoolVarP(&rebootYes, "yes", "y", false, "skip confirmation")

	brightnessCmd.AddCommand(brightnessUpCmd, brightnessDownCmd)

	tzCmd.AddCommand(tzSetCmd)

	rootCmd.AddCommand(rebootCmd, brightnessCmd, uptimeCmd)
}
