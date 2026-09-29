package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

const (
	configSnapshotFilePerm = 0o600
	unsetFieldDisplay      = "(unset)"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Device configuration snapshot: export, import, diff",
}

// FieldDiff is one section field whose value differs between the device and
// a saved snapshot, or is present on only one side.
type FieldDiff struct {
	Section string `json:"section"`
	Field   string `json:"field"`
	Device  any    `json:"device"`
	Saved   any    `json:"saved"`
}

// loadSnapshot reads and decodes a snapshot JSON file previously written by
// config export.
func loadSnapshot(path string) (*section.Snapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var s section.Snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("%s is not a valid config snapshot: %w", path, err)
	}
	return &s, nil
}

// snapshotMapField returns the map-valued section named name and whether it
// was present (non-nil) in s. It does not handle SectionTickers, which is a
// slice rather than a map.
func snapshotMapField(s *section.Snapshot, name string) (map[string]any, bool) {
	var m map[string]any
	switch name {
	case section.SectionDisplay:
		m = s.Display
	case section.SectionClock:
		m = s.Clock
	case section.SectionNTP:
		m = s.NTP
	case section.SectionWifi:
		m = s.Wifi
	case section.SectionAP:
		m = s.AP
	}
	return m, m != nil
}

// configSectionPresent reports whether s carries content for the named
// section.
func configSectionPresent(s *section.Snapshot, name string) bool {
	if name == section.SectionTickers {
		return s.Tickers != nil
	}
	_, ok := snapshotMapField(s, name)
	return ok
}

// configPresentSections returns the section names s carries content for, in
// canonical order.
func configPresentSections(s *section.Snapshot) []string {
	var names []string
	for _, name := range section.Sections {
		if configSectionPresent(s, name) {
			names = append(names, name)
		}
	}
	return names
}

// configIsSecretField reports whether field looks like a password or secret
// key, the same rule section.Capture applies when withSecrets is false.
func configIsSecretField(field string) bool {
	lower := strings.ToLower(field)
	return strings.Contains(lower, "password") || strings.Contains(lower, "secret")
}

// redactSnapshotSecrets returns a copy of s with password/secret fields
// removed from its wifi and ap sections. s itself is not mutated.
func redactSnapshotSecrets(s *section.Snapshot) *section.Snapshot {
	out := *s
	out.Wifi = redactMap(s.Wifi)
	out.AP = redactMap(s.AP)
	return &out
}

func redactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if configIsSecretField(k) {
			continue
		}
		out[k] = v
	}
	return out
}

// configDiffSnapshots compares every section device and saved carry, in
// canonical section order, and returns one FieldDiff per field that differs
// or is present on only one side.
func configDiffSnapshots(device, saved *section.Snapshot) []FieldDiff {
	var diffs []FieldDiff
	for _, name := range section.Sections {
		if name == section.SectionTickers {
			diffs = append(diffs, configDiffTickers(device.Tickers, saved.Tickers)...)
			continue
		}
		dm, _ := snapshotMapField(device, name)
		sm, _ := snapshotMapField(saved, name)
		diffs = append(diffs, diffMapSection(name, dm, sm)...)
	}
	return diffs
}

func diffMapSection(name string, device, saved map[string]any) []FieldDiff {
	keys := make(map[string]struct{}, len(device)+len(saved))
	for k := range device {
		keys[k] = struct{}{}
	}
	for k := range saved {
		keys[k] = struct{}{}
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var diffs []FieldDiff
	for _, k := range sorted {
		dv, dok := device[k]
		sv, sok := saved[k]
		if dok && sok && reflect.DeepEqual(dv, sv) {
			continue
		}
		diffs = append(diffs, FieldDiff{Section: name, Field: k, Device: fieldValue(dv, dok), Saved: fieldValue(sv, sok)})
	}
	return diffs
}

func fieldValue(v any, present bool) any {
	if !present {
		return nil
	}
	return v
}

// configDiffTickers compares two ticker lists positionally, reporting one
// FieldDiff per Entry field that differs, named tickers[i].<field>. An
// index present on only one side reports every field on that index as
// absent on the other.
func configDiffTickers(device, saved []tickers.Entry) []FieldDiff {
	n := len(device)
	if len(saved) > n {
		n = len(saved)
	}

	var diffs []FieldDiff
	for i := 0; i < n; i++ {
		dOK := i < len(device)
		sOK := i < len(saved)
		var d, s tickers.Entry
		if dOK {
			d = device[i]
		}
		if sOK {
			s = saved[i]
		}
		prefix := fmt.Sprintf("tickers[%d].", i)
		diffs = append(diffs, diffTickerField(prefix+"type", d.Type, s.Type, dOK, sOK)...)
		diffs = append(diffs, diffTickerField(prefix+"ticker", d.Ticker, s.Ticker, dOK, sOK)...)
		diffs = append(diffs, diffTickerField(prefix+"time", d.Time, s.Time, dOK, sOK)...)
		diffs = append(diffs, diffTickerField(prefix+"currency", d.Currency, s.Currency, dOK, sOK)...)
	}
	return diffs
}

func diffTickerField(field, device, saved string, dOK, sOK bool) []FieldDiff {
	if dOK && sOK && device == saved {
		return nil
	}
	if !dOK && !sOK {
		return nil
	}
	return []FieldDiff{{Section: section.SectionTickers, Field: field, Device: fieldValue(device, dOK), Saved: fieldValue(saved, sOK)}}
}

func configFormatDiffValue(v any) string {
	if v == nil {
		return unsetFieldDisplay
	}
	return fmt.Sprint(v)
}

var (
	configExportFile        string
	configExportShowSecrets bool
)

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Capture every device config section as a JSON snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := section.Capture(newClient(), section.Sections, configExportShowSecrets)
		if err != nil {
			return fmt.Errorf("capture device config: %w", err)
		}

		encoded, err := json.MarshalIndent(snap, "", "  ")
		if err != nil {
			return fmt.Errorf("encode snapshot: %w", err)
		}

		if configExportFile == "" {
			fmt.Println(string(encoded))
			return nil
		}
		if err := os.WriteFile(configExportFile, encoded, configSnapshotFilePerm); err != nil {
			return fmt.Errorf("write %s: %w", configExportFile, err)
		}
		fmt.Printf("wrote %s\n", configExportFile)
		return nil
	},
}

var configImportYes bool

var configImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Apply a saved config snapshot to the device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := loadSnapshot(args[0])
		if err != nil {
			return err
		}

		include := configPresentSections(snap)
		if len(include) == 0 {
			return fmt.Errorf("%s carries no config sections", args[0])
		}

		if !configImportYes && !confirm(fmt.Sprintf("Apply %s from %s to the device?", strings.Join(include, ", "), args[0])) {
			return fmt.Errorf("import aborted")
		}

		if err := section.Apply(newClient(), snap, include); err != nil {
			return fmt.Errorf("apply config: %w", err)
		}
		fmt.Printf("applied %s\n", strings.Join(include, ", "))
		return nil
	},
}

var configDiffShowSecrets bool

var configDiffCmd = &cobra.Command{
	Use:   "diff <file>",
	Short: "Show per-section field differences between the device and a saved snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		saved, err := loadSnapshot(args[0])
		if err != nil {
			return err
		}
		if !configDiffShowSecrets {
			saved = redactSnapshotSecrets(saved)
		}

		device, err := section.Capture(newClient(), section.Sections, configDiffShowSecrets)
		if err != nil {
			return fmt.Errorf("capture device config: %w", err)
		}

		diffs := configDiffSnapshots(device, saved)
		if jsonOut() {
			return output.EmitJSON(diffs)
		}
		if len(diffs) == 0 {
			fmt.Println("no differences")
			return nil
		}
		rows := make([][]string, len(diffs))
		for i, d := range diffs {
			rows[i] = []string{d.Section, d.Field, configFormatDiffValue(d.Device), configFormatDiffValue(d.Saved)}
		}
		output.Table([]string{"SECTION", "FIELD", "DEVICE", "SAVED"}, rows)
		return nil
	},
}

func init() {
	configExportCmd.Flags().StringVar(&configExportFile, "file", "", "file to write the snapshot to; defaults to stdout")
	configExportCmd.Flags().BoolVar(&configExportShowSecrets, "show-secrets", false, "include wifi/ap passwords and keys in the export")

	configImportCmd.Flags().BoolVarP(&configImportYes, "yes", "y", false, "skip confirmation")

	configDiffCmd.Flags().BoolVar(&configDiffShowSecrets, "show-secrets", false, "compare wifi/ap passwords and keys instead of redacting them")

	configCmd.AddCommand(configExportCmd, configImportCmd, configDiffCmd)
	rootCmd.AddCommand(configCmd)
}
