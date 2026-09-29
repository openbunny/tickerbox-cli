// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func fetchTickerEntries(c *client.Client) ([]tickers.Entry, error) {
	var state tickers.State
	if err := c.Get("coinSetupState", &state); err != nil {
		return nil, fmt.Errorf("get coinSetupState: %w", err)
	}
	entries, err := tickers.Decode(state)
	if err != nil {
		return nil, fmt.Errorf("decode coinSetupState: %w", err)
	}
	return entries, nil
}

func postTickerEntries(c *client.Client, entries []tickers.Entry) error {
	if err := c.Post("coinSetupState", tickers.Encode(entries)); err != nil {
		return fmt.Errorf("post coinSetupState: %w", err)
	}
	return nil
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
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
	Use:   "tickers",
	Short: "Ticker / asset list",
}

var tickersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured tickers",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := fetchTickerEntries(newClient())
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
		output.Table([]string{"#", "type", "ticker", "time", "currency"}, rows)
		return nil
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
		c := newClient()
		entries, err := fetchTickerEntries(c)
		if err != nil {
			return err
		}
		entries = append(entries, tickers.Entry{
			Type:     tickersAddType,
			Ticker:   tickersAddTicker,
			Time:     tickersAddTime,
			Currency: tickersAddCurrency,
		})
		if err := postTickerEntries(c, entries); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(entries)
		}
		fmt.Printf("Added %s\n", tickers.NormalizeTicker(tickersAddTicker))
		return nil
	},
}

var tickersRemoveCmd = &cobra.Command{
	Use:   "remove <index|ticker>",
	Short: "Remove a ticker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selector := args[0]
		c := newClient()
		entries, err := fetchTickerEntries(c)
		if err != nil {
			return err
		}

		var removed tickers.Entry
		if isIndex(selector) {
			idx, err := strconv.Atoi(selector)
			if err != nil || idx < 0 || idx >= len(entries) {
				return fmt.Errorf("index %s out of range: list has %d entries", selector, len(entries))
			}
			removed = entries[idx]
			entries = append(entries[:idx], entries[idx+1:]...)
		} else {
			target := tickers.NormalizeTicker(selector)
			found := slices.IndexFunc(entries, func(e tickers.Entry) bool {
				return tickers.NormalizeTicker(e.Ticker) == target
			})
			if found == -1 {
				return fmt.Errorf("no entry with ticker %s", target)
			}
			removed = entries[found]
			entries = append(entries[:found], entries[found+1:]...)
		}

		if err := postTickerEntries(c, entries); err != nil {
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
	Use:   "clear",
	Short: "Remove all tickers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !tickersClearYes && !confirm("Remove all tickers?") {
			return fmt.Errorf("clear aborted")
		}
		if err := postTickerEntries(newClient(), nil); err != nil {
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
	Use:   "export",
	Short: "Export tickers as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := fetchTickerEntries(newClient())
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
		if err := os.WriteFile(tickersExportOutput, encoded, 0o600); err != nil {
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
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := os.ReadFile(tickersImportInput)
		if err != nil {
			return fmt.Errorf("read %s: %w", tickersImportInput, err)
		}

		var entries []tickers.Entry
		if err := json.Unmarshal(raw, &entries); err != nil {
			return fmt.Errorf("%s is not a valid JSON array of ticker entries: %w", tickersImportInput, err)
		}

		if !tickersImportYes && !confirm(fmt.Sprintf("Replace the current list with %d entries from %s?", len(entries), tickersImportInput)) {
			return fmt.Errorf("import aborted")
		}

		if err := postTickerEntries(newClient(), entries); err != nil {
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

	tickersExportCmd.Flags().StringVar(&tickersExportOutput, "output", "", "file to write; defaults to stdout")

	tickersImportCmd.Flags().StringVar(&tickersImportInput, "input", "", "JSON file with an array of ticker entries")
	_ = tickersImportCmd.MarkFlagRequired("input")
	tickersImportCmd.Flags().BoolVarP(&tickersImportYes, "yes", "y", false, "skip confirmation")

	tickersCmd.AddCommand(tickersListCmd, tickersAddCmd, tickersRemoveCmd, tickersClearCmd, tickersExportCmd, tickersImportCmd)
	rootCmd.AddCommand(tickersCmd)
}
