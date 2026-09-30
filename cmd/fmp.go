// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/config"
)

var fmpCmd = &cobra.Command{
	Use:   "fmp",
	Short: "Financial Modeling Prep API key, used to verify ticker symbols on add",
	Long: "Manages the Financial Modeling Prep API key `ticker add` uses to verify a symbol before adding " +
		"it. Resolved from $TICKERBOX_FMP_API_KEY first, then the key stored here.",
}

var fmpSetKeyCmd = &cobra.Command{
	Use:     "set-key <key>",
	Short:   "Store a Financial Modeling Prep API key in the config file",
	Long:    "Persists KEY to config.toml. $TICKERBOX_FMP_API_KEY, when set, overrides the stored key without changing it.",
	Example: "  tickerbox fmp set-key abc123",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := cfg.SetFMPAPIKey(args[0]); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("set Financial Modeling Prep API key (%s)\n", maskSecret(args[0]))
		return nil
	},
}

func init() {
	fmpCmd.AddCommand(fmpSetKeyCmd)
	rootCmd.AddCommand(fmpCmd)
}
