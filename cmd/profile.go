// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/profile"
	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Named, device-agnostic device configurations",
}

var (
	profileSaveInclude string
	profileSaveAll     bool
)

var profileSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Capture the device's current config as a named profile",
	Long: "Captures tickers, display, clock, and ntp by default. --include selects specific sections by " +
		"name. --all also captures wifi and ap, including their passwords.",
	Example: "  tickerbox profile save home\n" +
		"  tickerbox profile save full --all",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		include, withSecrets := resolveInclude(profileSaveInclude, profileSaveAll)
		s, err := section.Capture(cmdContext(cmd), newClient(), include, withSecrets)
		if err != nil {
			return err
		}
		if err := profile.Save(args[0], s); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(s)
		}
		fmt.Printf("Saved profile %s (%s)\n", args[0], strings.Join(include, ","))
		return nil
	},
}

var profileListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List saved profiles",
	Example: "  tickerbox profile list",
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := profile.List()
		if err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(names)
		}
		for _, name := range names {
			fmt.Println(name)
		}
		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:     "show <name>",
	Short:   "Show a saved profile",
	Example: "  tickerbox profile show home",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := profile.Load(args[0])
		if err != nil {
			return err
		}
		return output.EmitJSON(s)
	},
}

var profileRmYes bool

var profileRmCmd = &cobra.Command{
	Use:     "rm <name>",
	Short:   "Delete a saved profile",
	Long:    "Prompts for confirmation unless --yes.",
	Example: "  tickerbox profile rm home --yes",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !profileRmYes && !confirm(fmt.Sprintf("Remove profile %s?", args[0])) {
			return fmt.Errorf("rm aborted")
		}
		if err := profile.Remove(args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed %s\n", args[0])
		return nil
	},
}

var profileApplyCmd = &cobra.Command{
	Use:   "apply <name>",
	Short: "Apply a saved profile to the device",
	Long: "Applies only the sections present in the saved profile; sections it doesn't contain are left " +
		"untouched on the device.",
	Example: "  tickerbox profile apply home",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := profile.Load(args[0])
		if err != nil {
			return err
		}
		include := presentSections(s)
		if err := section.Apply(cmdContext(cmd), newClient(), s, include); err != nil {
			return err
		}
		fmt.Printf("Applied profile %s (%s)\n", args[0], strings.Join(include, ","))
		return nil
	},
}

var profileDiffShowSecrets bool

var profileDiffCmd = &cobra.Command{
	Use:   "diff <name>",
	Short: "Compare a saved profile against the device's current config",
	Long: "Compares only the sections the profile contains. --show-secrets reveals wifi/ap password fields " +
		"instead of masking them.",
	Example: "  tickerbox profile diff home",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prof, err := profile.Load(args[0])
		if err != nil {
			return err
		}
		include := presentSections(prof)
		device, err := section.Capture(cmdContext(cmd), newClient(), include, profileDiffShowSecrets)
		if err != nil {
			return err
		}
		diffs := diffSnapshots(device, prof, include, profileDiffShowSecrets)

		if jsonOut() {
			return output.EmitJSON(diffs)
		}
		return renderDiffTable(diffs, []string{"SECTION", "FIELD", "DEVICE", "PROFILE"}, "No differences", func(d fieldDiff) []string {
			return []string{d.Section, d.Field, formatDiffValue(d.Device), formatDiffValue(d.Profile)}
		})
	},
}

func init() {
	profileSaveCmd.Flags().StringVar(&profileSaveInclude, "include", "", "comma-separated sections to capture (default "+strings.Join(section.DefaultInclude, ",")+")")
	profileSaveCmd.Flags().BoolVar(&profileSaveAll, "all", false, "also capture wifi and ap, including their secrets")

	profileRmCmd.Flags().BoolVarP(&profileRmYes, "yes", "y", false, "skip confirmation")

	profileDiffCmd.Flags().BoolVar(&profileDiffShowSecrets, "show-secrets", false, "reveal wifi/ap password fields instead of masking them")

	profileCmd.AddCommand(profileSaveCmd, profileListCmd, profileShowCmd, profileRmCmd, profileApplyCmd, profileDiffCmd)
	rootCmd.AddCommand(profileCmd)
}

func resolveInclude(includeFlag string, all bool) ([]string, bool) {
	include := section.DefaultInclude
	if includeFlag != "" {
		include = splitInclude(includeFlag)
	}
	if !all {
		return include, false
	}
	return appendMissing(include, section.SectionWifi, section.SectionAP), true
}

func splitInclude(raw string) []string {
	fields := strings.Split(raw, ",")
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func appendMissing(base []string, extra ...string) []string {
	out := append([]string{}, base...)
	for _, e := range extra {
		if !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out
}

func presentSections(s *section.Snapshot) []string {
	var out []string
	for _, name := range section.Sections {
		if sectionPresent(s, name) {
			out = append(out, name)
		}
	}
	return out
}

func sectionPresent(s *section.Snapshot, name string) bool {
	switch name {
	case section.SectionTickers:
		return len(s.Tickers) > 0
	case section.SectionDisplay:
		return s.Display != nil
	case section.SectionClock:
		return s.Clock != nil
	case section.SectionNTP:
		return s.NTP != nil
	case section.SectionWifi:
		return s.Wifi != nil
	case section.SectionAP:
		return s.AP != nil
	default:
		return false
	}
}

type fieldDiff struct {
	Section string `json:"section"`
	Field   string `json:"field"`
	Device  any    `json:"device"`
	Profile any    `json:"profile"`
}

func diffSnapshots(device, prof *section.Snapshot, sections []string, showSecrets bool) []fieldDiff {
	var diffs []fieldDiff
	for _, name := range sections {
		if name == section.SectionTickers {
			diffs = append(diffs, diffTickers(device.Tickers, prof.Tickers)...)
			continue
		}
		diffs = append(diffs, diffMap(name, sectionMap(device, name), sectionMap(prof, name), showSecrets)...)
	}
	return diffs
}

func sectionMap(s *section.Snapshot, name string) map[string]any {
	switch name {
	case section.SectionDisplay:
		return s.Display
	case section.SectionClock:
		return s.Clock
	case section.SectionNTP:
		return s.NTP
	case section.SectionWifi:
		return s.Wifi
	case section.SectionAP:
		return s.AP
	default:
		return nil
	}
}

func diffMap(name string, device, prof map[string]any, showSecrets bool) []fieldDiff {
	keySet := make(map[string]bool, len(device)+len(prof))
	for k := range device {
		keySet[k] = true
	}
	for k := range prof {
		keySet[k] = true
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var diffs []fieldDiff
	for _, k := range keys {
		dv, pv := device[k], prof[k]
		if reflect.DeepEqual(dv, pv) {
			continue
		}
		if !showSecrets && isSecretField(k) {
			dv, pv = maskValue(dv), maskValue(pv)
		}
		diffs = append(diffs, fieldDiff{Section: name, Field: k, Device: dv, Profile: pv})
	}
	return diffs
}

func isSecretField(field string) bool {
	lower := strings.ToLower(field)
	return strings.Contains(lower, "password") || strings.Contains(lower, "secret")
}

func maskValue(v any) any {
	if v == nil {
		return nil
	}
	return "********"
}

func diffTickers(device, prof []tickers.Entry) []fieldDiff {
	n := max(len(device), len(prof))
	var diffs []fieldDiff
	for i := 0; i < n; i++ {
		var d, p tickers.Entry
		if i < len(device) {
			d = device[i]
		}
		if i < len(prof) {
			p = prof[i]
		}
		if d == p {
			continue
		}
		diffs = append(diffs, fieldDiff{
			Section: section.SectionTickers,
			Field:   fmt.Sprintf("[%d]", i),
			Device:  entryOrNil(device, i),
			Profile: entryOrNil(prof, i),
		})
	}
	return diffs
}

func entryOrNil(entries []tickers.Entry, i int) any {
	if i >= len(entries) {
		return nil
	}
	return entries[i]
}

func formatDiffValue(v any) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%v", v)
}
