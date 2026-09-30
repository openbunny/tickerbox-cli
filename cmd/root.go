// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/config"
	"github.com/openbunny/tickerbox-cli/internal/output"
)

const (
	defaultRetries = 2
	defaultTimeout = 10 * time.Second

	maxSSIDLength = 32

	deviceRequestConcurrency = 3
	deviceFanOutConcurrency  = 8
)

var (
	hostFlag    string
	deviceFlag  string
	jsonFlag    bool
	retryFlag   int
	timeoutFlag time.Duration

	resolvedHost       string
	resolvedSource     config.HostSource
	resolvedDeviceName string
)

var rootCmd = &cobra.Command{
	Use:   "tickerbox",
	Short: "Control a TickerBox device over its REST API",
	Long: "tickerbox talks to a TickerBox ESP32 device's REST API over HTTP. Target a device with --host, " +
		"or register named devices with `tickerbox device add` and select one with --device or `tickerbox device use`. " +
		"The device's REST API has no authentication; anything on its network can reach it.",
	Example: "  tickerbox device add desk http://tickerbox.local\n" +
		"  tickerbox status --device desk\n" +
		"  tickerbox --host http://192.168.1.42 wifi status",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !commandTargetsDevice(cmd) {
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("resolve target host: %w", err)
		}
		resolved, err := cfg.Resolve(hostFlag, deviceFlag)
		if err != nil {
			return fmt.Errorf("resolve target host: %w", err)
		}
		resolvedHost = resolved.Host
		resolvedSource = resolved.Source
		resolvedDeviceName = resolved.DeviceName

		if !jsonOut() && resolved.DeviceName != "" && len(cfg.Devices) > 1 {
			return output.DeviceBanner(resolved.DeviceName, resolved.Host)
		}
		return nil
	},
}

// deviceIndependentCommands holds every command whose own subtree never resolves a
// target device (no newClient() call, no read of resolvedHost/resolvedSource/
// resolvedDeviceName anywhere in its RunE), found by auditing every cmd/*.go file.
// Membership is by exact command, not just top-level groups: profileListCmd,
// profileShowCmd, and profileRmCmd are excluded individually because their sibling
// leaves profileSaveCmd/profileApplyCmd/profileDiffCmd do need a resolved device.
var deviceIndependentCommands = map[*cobra.Command]bool{}

func commandTargetsDevice(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if deviceIndependentCommands[c] {
			return false
		}
	}
	return true
}

func init() {
	rootCmd.PersistentFlags().StringVar(&hostFlag, "host", "", "TickerBox base URL (default "+client.DefaultHost+", or $TICKERBOX_HOST)")
	rootCmd.PersistentFlags().StringVarP(&deviceFlag, "device", "d", "", "named device from the config file")
	rootCmd.PersistentFlags().BoolVarP(&jsonFlag, "json", "j", false, "emit raw device JSON instead of a decoded table")
	rootCmd.PersistentFlags().IntVar(&retryFlag, "retry", defaultRetries, "retry an idempotent request this many times on a transient failure")
	rootCmd.PersistentFlags().DurationVar(&timeoutFlag, "timeout", defaultTimeout, "per-request timeout")

	rootCmd.SilenceErrors = true

	deviceIndependentCommands[deviceCmd] = true
	deviceIndependentCommands[tzCmd] = true
	deviceIndependentCommands[profileListCmd] = true
	deviceIndependentCommands[profileShowCmd] = true
	deviceIndependentCommands[profileRmCmd] = true
	deviceIndependentCommands[versionCmd] = true
}

func Execute() error {
	return rootCmd.Execute()
}

func Root() *cobra.Command {
	return rootCmd
}

func restBase(host string) string {
	return strings.TrimRight(host, "/") + "/rest/"
}

func newClient() *client.Client {
	c := client.New(restBase(resolvedHost), timeoutFlag)
	c.Retries = retryFlag
	return c
}

func jsonOut() bool {
	return jsonFlag
}

func cmdContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}
