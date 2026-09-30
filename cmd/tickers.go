// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func fetchTickerEntries(ctx context.Context, c *client.Client) ([]tickers.Entry, error) {
	var state tickers.State
	if err := c.Get(ctx, "coinSetupState", &state); err != nil {
		return nil, fmt.Errorf("get coinSetupState: %w", err)
	}
	entries, err := tickers.Decode(state)
	if err != nil {
		return nil, fmt.Errorf("decode coinSetupState: %w", err)
	}
	return entries, nil
}

func postTickerEntries(ctx context.Context, c *client.Client, entries []tickers.Entry) error {
	encoded, err := tickers.Encode(entries)
	if err != nil {
		return fmt.Errorf("encode ticker entries: %w", err)
	}
	if err := c.Post(ctx, "coinSetupState", encoded); err != nil {
		return fmt.Errorf("post coinSetupState: %w", err)
	}
	return nil
}

var confirmStdin io.Reader = os.Stdin

func confirm(prompt string) (bool, error) {
	fmt.Printf("%s [y/N]: ", prompt)
	line, err := bufio.NewReader(confirmStdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

func isIndex(selector string) bool {
	if selector == "" {
		return false
	}
	for _, r := range selector {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

var tickersCmd = &cobra.Command{
	Use:   "ticker",
	Short: "Ticker / asset list",
}

var tickersListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List configured tickers",
	Example: "  tickerbox ticker list",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := fetchTickerEntries(cmdContext(cmd), newClient())
		if err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(entries)
		}
		rows := make([][]string, len(entries))
		for i, e := range entries {
			rows[i] = []string{strconv.Itoa(i), e.Type, e.Ticker, e.Time, e.Currency}
		}
		return output.Table([]string{"#", "type", "ticker", "time", "currency"}, rows)
	},
}

var (
	tickersAddType     string
	tickersAddTicker   string
	tickersAddTime     string
	tickersAddCurrency string
)

var tickersAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a ticker",
	RunE: func(cmd *cobra.Command, args []string) error {
		newEntry := tickers.Entry{
			Type:     tickersAddType,
			Ticker:   tickersAddTicker,
			Time:     tickersAddTime,
			Currency: tickersAddCurrency,
		}
		if err := rejectInvalidEntries([]tickers.Entry{newEntry}); err != nil {
			return err
		}
		c := newClient()
		entries, err := fetchTickerEntries(cmdContext(cmd), c)
		if err != nil {
			return err
		}
		entries = append(entries, newEntry)
		if err := rejectInvalidEntries(entries); err != nil {
			return err
		}
		if err := postTickerEntries(cmdContext(cmd), c, entries); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(entries)
		}
		fmt.Printf("Added %s\n", tickers.NormalizeTicker(tickersAddTicker))
		return nil
	},
}

var tickersRemoveYes bool

var tickersRemoveCmd = &cobra.Command{
	Use:   "rm <index|ticker>",
	Short: "Remove a ticker",
	Long:  "Accepts either the 0-based index shown by `ticker list`, or a ticker symbol. Prompts for confirmation unless --yes.",
	Example: "  tickerbox ticker rm BTC\n" +
		"  tickerbox ticker rm 0",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selector := args[0]
		c := newClient()
		entries, err := fetchTickerEntries(cmdContext(cmd), c)
		if err != nil {
			return err
		}

		idx, err := resolveEntryIndex(entries, selector)
		if err != nil {
			return err
		}
		removed := entries[idx]
		entries = append(entries[:idx], entries[idx+1:]...)

		if !tickersRemoveYes {
			ok, err := confirm(fmt.Sprintf("Remove ticker %s?", removed.Ticker))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}

		if err := postTickerEntries(cmdContext(cmd), c, entries); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(entries)
		}
		fmt.Printf("Removed %s\n", removed.Ticker)
		return nil
	},
}

var tickersClearYes bool

var tickersClearCmd = &cobra.Command{
	Use:     "clear",
	Short:   "Remove all tickers",
	Long:    "Removes every ticker. Prompts for confirmation unless --yes.",
	Example: "  tickerbox ticker clear --yes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !tickersClearYes {
			ok, err := confirm("Remove all tickers?")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := postTickerEntries(cmdContext(cmd), newClient(), nil); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON([]tickers.Entry{})
		}
		fmt.Println("Cleared")
		return nil
	},
}

var tickersExportOutput string

var tickersExportCmd = &cobra.Command{
	Use:     "export",
	Short:   "Export tickers as JSON",
	Example: "  tickerbox ticker export --output tickers.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := fetchTickerEntries(cmdContext(cmd), newClient())
		if err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("encode entries: %w", err)
		}
		if tickersExportOutput == "" {
			fmt.Println(string(encoded))
			return nil
		}
		if err := os.WriteFile(tickersExportOutput, encoded, configSnapshotFilePerm); err != nil {
			return fmt.Errorf("write %s: %w", tickersExportOutput, err)
		}
		return nil
	},
}

var (
	tickersImportInput string
	tickersImportYes   bool
)

var tickersImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Replace tickers from a JSON file",
	Long: "Replaces the entire ticker list with the contents of the file; existing entries not present in " +
		"the file are dropped. Prompts for confirmation unless --yes.",
	Example: "  tickerbox ticker import --input tickers.json --yes",
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := os.ReadFile(tickersImportInput)
		if err != nil {
			return fmt.Errorf("read %s: %w", tickersImportInput, err)
		}

		var entries []tickers.Entry
		if err := json.Unmarshal(raw, &entries); err != nil {
			return fmt.Errorf("%s is not a valid JSON array of ticker entries: %w", tickersImportInput, err)
		}
		if err := rejectInvalidEntries(entries); err != nil {
			return fmt.Errorf("%s: %w", tickersImportInput, err)
		}

		if !tickersImportYes {
			ok, err := confirm(fmt.Sprintf("Replace the current list with %d entries from %s?", len(entries), tickersImportInput))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}

		if err := postTickerEntries(cmdContext(cmd), newClient(), entries); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(entries)
		}
		fmt.Printf("Imported %d entries\n", len(entries))
		return nil
	},
}

func init() {
	tickersAddCmd.Flags().StringVar(&tickersAddType, "type", "", "crypto|stocks|forex")
	tickersAddCmd.Flags().StringVar(&tickersAddTicker, "ticker", "", "ticker symbol")
	tickersAddCmd.Flags().StringVar(&tickersAddTime, "time", "", "1min|5min|15min")
	tickersAddCmd.Flags().StringVar(&tickersAddCurrency, "currency", "USD", "USD|EUR|GBP|CAD|AUD|JPY")
	_ = tickersAddCmd.MarkFlagRequired("type")
	_ = tickersAddCmd.MarkFlagRequired("ticker")
	_ = tickersAddCmd.MarkFlagRequired("time")

	tickersClearCmd.Flags().BoolVarP(&tickersClearYes, "yes", "y", false, "skip confirmation")

	tickersRemoveCmd.Flags().BoolVarP(&tickersRemoveYes, "yes", "y", false, "skip confirmation")

	tickersExportCmd.Flags().StringVar(&tickersExportOutput, "output", "", "file to write; defaults to stdout")

	tickersImportCmd.Flags().StringVar(&tickersImportInput, "input", "", "JSON file with an array of ticker entries")
	_ = tickersImportCmd.MarkFlagRequired("input")
	tickersImportCmd.Flags().BoolVarP(&tickersImportYes, "yes", "y", false, "skip confirmation")

	tickersCmd.AddCommand(tickersListCmd, tickersAddCmd, tickersRemoveCmd, tickersClearCmd, tickersExportCmd, tickersImportCmd)
	rootCmd.AddCommand(tickersCmd)
}
