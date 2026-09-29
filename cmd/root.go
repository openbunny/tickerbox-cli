package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/config"
)

const (
	defaultRetries = 2
	defaultTimeout = 10 * time.Second
)

var (
	hostFlag    string
	deviceFlag  string
	jsonFlag    bool
	retryFlag   int
	timeoutFlag time.Duration

	resolvedHost string
)

var rootCmd = &cobra.Command{
	Use:   "tickerbox",
	Short: "Control a TickerBox device over its REST API",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		host, err := config.ResolveHost(hostFlag, deviceFlag)
		if err != nil {
			return fmt.Errorf("resolve target host: %w", err)
		}
		resolvedHost = host
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&hostFlag, "host", "", "TickerBox base URL (default "+client.DefaultHost+", or $TICKERBOX_HOST)")
	rootCmd.PersistentFlags().StringVarP(&deviceFlag, "device", "d", "", "named device from the config file")
	rootCmd.PersistentFlags().BoolVarP(&jsonFlag, "json", "j", false, "emit raw device JSON instead of a decoded table")
	rootCmd.PersistentFlags().IntVar(&retryFlag, "retry", defaultRetries, "retry an idempotent request this many times on a transient failure")
	rootCmd.PersistentFlags().DurationVar(&timeoutFlag, "timeout", defaultTimeout, "per-request timeout")
}

// Execute runs the root command, resolving the target device host in its
// PersistentPreRunE before any subcommand runs.
func Execute() error {
	return rootCmd.Execute()
}

// restBase turns a resolved device host into the client's base URL.
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
