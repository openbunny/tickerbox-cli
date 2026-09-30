// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/config"
	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/template"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func resolveEntryIndex(entries []tickers.Entry, selector string) (int, error) {
	if isIndex(selector) {
		idx, err := strconv.Atoi(selector)
		if err != nil || idx < 0 || idx >= len(entries) {
			return -1, fmt.Errorf("index %s out of range: list has %d entries", selector, len(entries))
		}
		return idx, nil
	}
	target := tickers.NormalizeTicker(selector)
	if idx := slices.IndexFunc(entries, func(e tickers.Entry) bool {
		return tickers.NormalizeTicker(e.Ticker) == target
	}); idx != -1 {
		return idx, nil
	}
	return -1, fmt.Errorf("no entry with ticker %s", target)
}

func buildBulkEntries(symbols []string, typ, tm, currency string) []tickers.Entry {
	entries := make([]tickers.Entry, len(symbols))
	for i, sym := range symbols {
		entries[i] = tickers.Entry{Type: typ, Ticker: sym, Time: tm, Currency: currency}
	}
	return entries
}

type editFields struct {
	ticker      string
	tickerSet   bool
	typ         string
	typeSet     bool
	time        string
	timeSet     bool
	currency    string
	currencySet bool
}

func applyEdit(e tickers.Entry, f editFields) tickers.Entry {
	if f.tickerSet {
		e.Ticker = tickers.NormalizeTicker(f.ticker)
	}
	if f.typeSet {
		e.Type = f.typ
	}
	if f.timeSet {
		e.Time = f.time
	}
	if f.currencySet {
		e.Currency = f.currency
	}
	return e
}

func reorder(entries []tickers.Entry, from, to int) ([]tickers.Entry, error) {
	n := len(entries)
	if from < 0 || from >= n {
		return nil, fmt.Errorf("from index %d out of range: list has %d entries", from, n)
	}
	if to < 0 || to >= n {
		return nil, fmt.Errorf("to index %d out of range: list has %d entries", to, n)
	}

	moved := entries[from]
	rest := make([]tickers.Entry, 0, n-1)
	for i, e := range entries {
		if i != from {
			rest = append(rest, e)
		}
	}

	out := make([]tickers.Entry, 0, n)
	out = append(out, rest[:to]...)
	out = append(out, moved)
	out = append(out, rest[to:]...)
	return out, nil
}

func validateEntries(entries []tickers.Entry) []string {
	return tickers.Validate(entries)
}

func rejectInvalidEntries(entries []tickers.Entry) error {
	problems := validateEntries(entries)
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("invalid ticker entries:\n%s", strings.Join(problems, "\n"))
}

func rejectInvalidEditFlags(cmd *cobra.Command, typ, tm, currency string) error {
	flags := cmd.Flags()
	if flags.Changed("type") && !tickers.ValidType(typ) {
		return fmt.Errorf("invalid --type %q: expected one of %s", typ, tickers.ExpectedTypes())
	}
	if flags.Changed("time") && !tickers.ValidTime(tm) {
		return fmt.Errorf("invalid --time %q: expected one of %s", tm, tickers.ExpectedTimes())
	}
	if flags.Changed("currency") && !tickers.ValidCurrency(currency) {
		return fmt.Errorf("invalid --currency %q: expected one of %s", currency, tickers.ExpectedCurrencies())
	}
	return nil
}

// fmpVerifyTimeout is the FMP request timeout, fixed at defaultTimeout (not the
// device's --timeout flag) since verification targets an unrelated host. A test
// override needs a var rather than defaultTimeout's const.
var fmpVerifyTimeout = defaultTimeout

// verifyNewTickers checks each entry's ticker against Financial Modeling Prep before
// it reaches the device. A verification failure (missing key, unreachable API) is
// never treated as an unknown symbol — the two are reported with distinct error
// text so a transient FMP outage can't silently reject a real ticker.
func verifyNewTickers(ctx context.Context, entries []tickers.Entry) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	apiKey, ok := cfg.ResolveFMPAPIKey()
	if !ok {
		return fmt.Errorf("verify tickers: no Financial Modeling Prep API key configured — set %s, or pass --no-verify to add without verification",
			config.EnvFMPAPIKey)
	}

	fmp := client.NewFMP(fmpVerifyTimeout)
	for _, e := range entries {
		symbol := tickers.NormalizeTicker(e.Ticker)
		known, err := fmp.Verify(ctx, symbol, apiKey)
		if err != nil {
			return fmt.Errorf("verify ticker %q: Financial Modeling Prep unreachable: %w — retry, or pass --no-verify to add without verification", symbol, err)
		}
		if !known {
			return fmt.Errorf("ticker %q: not found on Financial Modeling Prep — check the symbol (e.g. AAPL, BTC, EURUSD) or pass --no-verify to add it unchecked", symbol)
		}
	}
	return nil
}

func applyTemplatePreset(existing, preset []tickers.Entry, replace bool) []tickers.Entry {
	if replace {
		out := make([]tickers.Entry, len(preset))
		copy(out, preset)
		return out
	}

	have := make(map[string]bool, len(existing))
	for _, e := range existing {
		have[tickers.NormalizeTicker(e.Ticker)] = true
	}

	out := make([]tickers.Entry, len(existing), len(existing)+len(preset))
	copy(out, existing)
	for _, e := range preset {
		norm := tickers.NormalizeTicker(e.Ticker)
		if have[norm] {
			continue
		}
		have[norm] = true
		out = append(out, e)
	}
	return out
}

var (
	tickersAddBulkType     string
	tickersAddBulkTime     string
	tickersAddBulkCurrency string
	tickersAddBulkNoVerify bool
)

var tickersAddBulkCmd = &cobra.Command{
	Use:   "add <sym>...",
	Short: "Add one or more tickers, sharing --type/--time/--currency",
	Long: "Adds one entry per sym, all sharing the same --type, --time, and --currency. --type and --time " +
		"are required; --currency defaults to USD.",
	Example: "  tickerbox ticker add BTC ETH --type crypto --time 5min --currency USD",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newEntries := buildBulkEntries(args, tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency)
		if err := rejectInvalidEntries(newEntries); err != nil {
			return err
		}
		if !tickersAddBulkNoVerify {
			if err := verifyNewTickers(cmdContext(cmd), newEntries); err != nil {
				return err
			}
		}
		c := newClient()
		entries, err := fetchTickerEntries(cmdContext(cmd), c)
		if err != nil {
			return err
		}
		entries = append(entries, newEntries...)
		if err := rejectInvalidEntries(entries); err != nil {
			return err
		}
		if err := postTickerEntries(cmdContext(cmd), c, entries); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(entries)
		}
		fmt.Printf("Added %d ticker(s)\n", len(args))
		return nil
	},
}

var (
	tickersEditTicker   string
	tickersEditType     string
	tickersEditTime     string
	tickersEditCurrency string
	tickersEditYes      bool
)

var tickersEditCmd = &cobra.Command{
	Use:     "edit <index|sym>",
	Short:   "Change fields of one ticker entry in place",
	Example: "  tickerbox ticker edit BTC --time 1min",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := rejectInvalidEditFlags(cmd, tickersEditType, tickersEditTime, tickersEditCurrency); err != nil {
			return err
		}
		c := newClient()
		entries, err := fetchTickerEntries(cmdContext(cmd), c)
		if err != nil {
			return err
		}
		idx, err := resolveEntryIndex(entries, args[0])
		if err != nil {
			return err
		}
		original := entries[idx]
		entries[idx] = applyEdit(entries[idx], editFields{
			ticker:      tickersEditTicker,
			tickerSet:   cmd.Flags().Changed("ticker"),
			typ:         tickersEditType,
			typeSet:     cmd.Flags().Changed("type"),
			time:        tickersEditTime,
			timeSet:     cmd.Flags().Changed("time"),
			currency:    tickersEditCurrency,
			currencySet: cmd.Flags().Changed("currency"),
		})
		if err := rejectInvalidEntries(entries); err != nil {
			return err
		}
		if !tickersEditYes {
			ok, err := confirm(fmt.Sprintf("Overwrite ticker %s with the new fields?", original.Ticker))
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
		fmt.Printf("Edited %s\n", entries[idx].Ticker)
		return nil
	},
}

var tickersMoveCmd = &cobra.Command{
	Use:   "move <from> <to>",
	Short: "Reorder the ticker list; the display cycles in this order",
	Long: "FROM and TO are 0-based positions in the order the display cycles through, as shown by " +
		"`ticker list`.",
	Example: "  tickerbox ticker move 3 0",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		from, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("from index %q is not a number", args[0])
		}
		to, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("to index %q is not a number", args[1])
		}

		c := newClient()
		entries, err := fetchTickerEntries(cmdContext(cmd), c)
		if err != nil {
			return err
		}
		reordered, err := reorder(entries, from, to)
		if err != nil {
			return err
		}
		if err := postTickerEntries(cmdContext(cmd), c, reordered); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(reordered)
		}
		fmt.Printf("Moved entry %d to position %d\n", from, to)
		return nil
	},
}

var tickersValidateCmd = &cobra.Command{
	Use:     "validate",
	Short:   "Report illegal fields and duplicate symbols in the ticker list",
	Example: "  tickerbox ticker validate",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := fetchTickerEntries(cmdContext(cmd), newClient())
		if err != nil {
			return err
		}
		problems := validateEntries(entries)
		if jsonOut() {
			return output.EmitJSON(problems)
		}
		if len(problems) == 0 {
			fmt.Println("No problems found")
			return nil
		}
		for _, p := range problems {
			fmt.Println(p)
		}
		return fmt.Errorf("%d problem(s) found", len(problems))
	},
}

var tickersTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Built-in ticker list presets",
}

type presetSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var tickersTemplateListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List built-in preset names",
	Example: "  tickerbox ticker template list",
	RunE: func(cmd *cobra.Command, args []string) error {
		names := template.List()
		summaries := make([]presetSummary, len(names))
		for i, n := range names {
			desc, _ := template.Describe(n)
			summaries[i] = presetSummary{Name: n, Description: desc}
		}
		if jsonOut() {
			return output.EmitJSON(summaries)
		}
		rows := make([][]string, len(summaries))
		for i, s := range summaries {
			rows[i] = []string{s.Name, s.Description}
		}
		return output.Table([]string{"name", "description"}, rows)
	},
}

type presetShow struct {
	Description string          `json:"description"`
	Entries     []tickers.Entry `json:"entries"`
}

var tickersTemplateShowCmd = &cobra.Command{
	Use:     "show <name>",
	Short:   "Show a preset's entries",
	Example: "  tickerbox ticker template show crypto-top10",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, ok := template.Get(args[0])
		if !ok {
			return fmt.Errorf("no such template %q", args[0])
		}
		desc, _ := template.Describe(args[0])
		if jsonOut() {
			return output.EmitJSON(presetShow{Description: desc, Entries: entries})
		}
		fmt.Println(desc)
		rows := make([][]string, len(entries))
		for i, e := range entries {
			rows[i] = []string{strconv.Itoa(i), e.Type, e.Ticker, e.Time, e.Currency}
		}
		return output.Table([]string{"#", "type", "ticker", "time", "currency"}, rows)
	},
}

var (
	tickersTemplateReplace  bool
	tickersTemplateApplyYes bool
)

var tickersTemplateApplyCmd = &cobra.Command{
	Use:   "apply <name>",
	Short: "Apply a preset to the current ticker list",
	Long: "By default appends the preset's entries, skipping any ticker already in the list. --replace " +
		"discards the current list and uses the preset's entries only.",
	Example: "  tickerbox ticker template apply crypto-top10 --replace",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		preset, ok := template.Get(args[0])
		if !ok {
			return fmt.Errorf("no such template %q", args[0])
		}
		if err := rejectInvalidEntries(preset); err != nil {
			return err
		}
		c := newClient()
		existing, err := fetchTickerEntries(cmdContext(cmd), c)
		if err != nil {
			return err
		}
		merged := applyTemplatePreset(existing, preset, tickersTemplateReplace)
		if err := rejectInvalidEntries(merged); err != nil {
			return err
		}
		if tickersTemplateReplace && !tickersTemplateApplyYes {
			ok, err := confirm(fmt.Sprintf("Replace the current list with %d entries from preset %s?", len(preset), args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := postTickerEntries(cmdContext(cmd), c, merged); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(merged)
		}
		fmt.Printf("Applied template %s: %d entries\n", args[0], len(merged))
		return nil
	},
}

func init() {
	tickersAddBulkCmd.Flags().StringVar(&tickersAddBulkType, "type", "", "crypto|stocks|forex")
	tickersAddBulkCmd.Flags().StringVar(&tickersAddBulkTime, "time", "", "1min|5min|15min")
	tickersAddBulkCmd.Flags().StringVar(&tickersAddBulkCurrency, "currency", tickers.CurrencyUSD, "USD|EUR|GBP|CAD|AUD|JPY")
	tickersAddBulkCmd.Flags().BoolVar(&tickersAddBulkNoVerify, "no-verify", false, "skip Financial Modeling Prep symbol verification, add the entry unchecked")
	_ = tickersAddBulkCmd.MarkFlagRequired("type")
	_ = tickersAddBulkCmd.MarkFlagRequired("time")

	tickersEditCmd.Flags().StringVar(&tickersEditTicker, "ticker", "", "new ticker symbol")
	tickersEditCmd.Flags().StringVar(&tickersEditType, "type", "", "crypto|stocks|forex")
	tickersEditCmd.Flags().StringVar(&tickersEditTime, "time", "", "1min|5min|15min")
	tickersEditCmd.Flags().StringVar(&tickersEditCurrency, "currency", "", "USD|EUR|GBP|CAD|AUD|JPY")
	tickersEditCmd.Flags().BoolVarP(&tickersEditYes, "yes", "y", false, "skip confirmation")

	tickersTemplateApplyCmd.Flags().BoolVar(&tickersTemplateReplace, "replace", false, "replace the current list instead of appending")
	var tickersTemplateAppendFlag bool
	tickersTemplateApplyCmd.Flags().BoolVar(&tickersTemplateAppendFlag, "append", true, "append to the current list, skipping duplicates (default)")
	tickersTemplateApplyCmd.MarkFlagsMutuallyExclusive("append", "replace")
	tickersTemplateApplyCmd.Flags().BoolVarP(&tickersTemplateApplyYes, "yes", "y", false, "skip confirmation")

	tickersTemplateCmd.AddCommand(tickersTemplateListCmd, tickersTemplateShowCmd, tickersTemplateApplyCmd)

	tickersCmd.RemoveCommand(tickersAddCmd)
	tickersCmd.AddCommand(tickersAddBulkCmd, tickersEditCmd, tickersMoveCmd, tickersValidateCmd, tickersTemplateCmd)
}
