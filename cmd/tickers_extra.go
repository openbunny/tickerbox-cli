package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/template"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

// resolveEntryIndex resolves selector, an index or a ticker symbol, to its
// position in entries.
func resolveEntryIndex(entries []tickers.Entry, selector string) (int, error) {
	if isIndex(selector) {
		idx, err := strconv.Atoi(selector)
		if err != nil || idx < 0 || idx >= len(entries) {
			return -1, fmt.Errorf("index %s out of range: list has %d entries", selector, len(entries))
		}
		return idx, nil
	}
	target := tickers.NormalizeTicker(selector)
	for i, e := range entries {
		if tickers.NormalizeTicker(e.Ticker) == target {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no entry with ticker %s", target)
}

// buildBulkEntries builds one Entry per symbol, all sharing typ, tm and
// currency.
func buildBulkEntries(symbols []string, typ, tm, currency string) []tickers.Entry {
	entries := make([]tickers.Entry, len(symbols))
	for i, sym := range symbols {
		entries[i] = tickers.Entry{Type: typ, Ticker: sym, Time: tm, Currency: currency}
	}
	return entries
}

// editFields holds the fields an edit command may change and whether each
// was actually set on the command line, so applyEdit only touches fields the
// caller named.
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

// applyEdit returns e with every field named by f overwritten.
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

// reorder returns entries with the one at from moved to position to, other
// entries shifting to keep the list contiguous. from and to are positions in
// entries, the order the device's display cycles through.
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

// validateEntries reports every illegal type, time or currency value and
// every duplicate ticker symbol in entries, one message per problem.
func validateEntries(entries []tickers.Entry) []string {
	var problems []string
	seen := make(map[string]int, len(entries))
	for i, e := range entries {
		if !tickers.ValidType(e.Type) {
			problems = append(problems, fmt.Sprintf("entry %d (%s): invalid type %q", i, e.Ticker, e.Type))
		}
		if !tickers.ValidTime(e.Time) {
			problems = append(problems, fmt.Sprintf("entry %d (%s): invalid time %q", i, e.Ticker, e.Time))
		}
		if !tickers.ValidCurrency(e.Currency) {
			problems = append(problems, fmt.Sprintf("entry %d (%s): invalid currency %q", i, e.Ticker, e.Currency))
		}
		norm := tickers.NormalizeTicker(e.Ticker)
		if first, ok := seen[norm]; ok {
			problems = append(problems, fmt.Sprintf("entry %d (%s): duplicate of entry %d", i, e.Ticker, first))
			continue
		}
		seen[norm] = i
	}
	return problems
}

// applyTemplatePreset returns the entries a template apply should post:
// preset alone when replace is true, otherwise existing with every preset
// entry appended whose normalized ticker is not already present.
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
)

var tickersAddBulkCmd = &cobra.Command{
	Use:   "add SYM...",
	Short: "Add one or more tickers, sharing --type/--time/--currency",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		entries, err := fetchTickerEntries(c)
		if err != nil {
			return err
		}
		entries = append(entries, buildBulkEntries(args, tickersAddBulkType, tickersAddBulkTime, tickersAddBulkCurrency)...)
		if err := postTickerEntries(c, entries); err != nil {
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
)

var tickersEditCmd = &cobra.Command{
	Use:   "edit <index|sym>",
	Short: "Change fields of one ticker entry in place",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		entries, err := fetchTickerEntries(c)
		if err != nil {
			return err
		}
		idx, err := resolveEntryIndex(entries, args[0])
		if err != nil {
			return err
		}
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
		if err := postTickerEntries(c, entries); err != nil {
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
	Args:  cobra.ExactArgs(2),
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
		entries, err := fetchTickerEntries(c)
		if err != nil {
			return err
		}
		reordered, err := reorder(entries, from, to)
		if err != nil {
			return err
		}
		if err := postTickerEntries(c, reordered); err != nil {
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
	Use:   "validate",
	Short: "Report illegal fields and duplicate symbols in the ticker list",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := fetchTickerEntries(newClient())
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

var tickersTemplateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List built-in preset names",
	RunE: func(cmd *cobra.Command, args []string) error {
		names := template.List()
		if jsonOut() {
			return output.EmitJSON(names)
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return nil
	},
}

var tickersTemplateShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a preset's entries",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, ok := template.Get(args[0])
		if !ok {
			return fmt.Errorf("no such template %q", args[0])
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

var tickersTemplateReplace bool

var tickersTemplateApplyCmd = &cobra.Command{
	Use:   "apply <name>",
	Short: "Apply a preset to the current ticker list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		preset, ok := template.Get(args[0])
		if !ok {
			return fmt.Errorf("no such template %q", args[0])
		}
		c := newClient()
		existing, err := fetchTickerEntries(c)
		if err != nil {
			return err
		}
		merged := applyTemplatePreset(existing, preset, tickersTemplateReplace)
		if err := postTickerEntries(c, merged); err != nil {
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
	_ = tickersAddBulkCmd.MarkFlagRequired("type")
	_ = tickersAddBulkCmd.MarkFlagRequired("time")

	tickersEditCmd.Flags().StringVar(&tickersEditTicker, "ticker", "", "new ticker symbol")
	tickersEditCmd.Flags().StringVar(&tickersEditType, "type", "", "crypto|stocks|forex")
	tickersEditCmd.Flags().StringVar(&tickersEditTime, "time", "", "1min|5min|15min")
	tickersEditCmd.Flags().StringVar(&tickersEditCurrency, "currency", "", "USD|EUR|GBP|CAD|AUD|JPY")

	tickersTemplateApplyCmd.Flags().BoolVar(&tickersTemplateReplace, "replace", false, "replace the current list instead of appending")
	var tickersTemplateAppendFlag bool
	tickersTemplateApplyCmd.Flags().BoolVar(&tickersTemplateAppendFlag, "append", true, "append to the current list, skipping duplicates (default)")
	tickersTemplateApplyCmd.MarkFlagsMutuallyExclusive("append", "replace")

	tickersTemplateCmd.AddCommand(tickersTemplateListCmd, tickersTemplateShowCmd, tickersTemplateApplyCmd)

	// tickersAddCmd (tickers.go) takes one ticker via --ticker; this bulk,
	// positional-args form replaces it under the same "add" name.
	tickersCmd.RemoveCommand(tickersAddCmd)
	tickersCmd.AddCommand(tickersAddBulkCmd, tickersEditCmd, tickersMoveCmd, tickersValidateCmd, tickersTemplateCmd)
}
