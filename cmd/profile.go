// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/diff"
	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/profile"
	"github.com/openbunny/tickerbox-cli/internal/section"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Named, device-agnostic device configurations",
	Long: "Named, device-agnostic bundles of ticker/display/clock/ntp settings (and, with --all, wifi/ap " +
		"secrets), applied to whichever device currently resolves — see `tickerbox device current`. Distinct " +
		"from `tickerbox config`, which exports and diffs one device's full live snapshot rather than a " +
		"saved, reusable bundle.",
}

var (
	profileSaveInclude     string
	profileSaveAll         bool
	profileSaveDescription string
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
		if err := profile.SaveDescribed(args[0], s, profileSaveDescription, resolvedDeviceName); err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(s)
		}
		fmt.Printf("Saved profile %s (%s)\n", args[0], strings.Join(include, ","))
		return nil
	},
}

type profileListEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
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
		entries := make([]profileListEntry, 0, len(names))
		for _, name := range names {
			description, err := profile.LoadDescription(name)
			if err != nil {
				return err
			}
			entries = append(entries, profileListEntry{Name: name, Description: description})
		}
		if jsonOut() {
			return output.EmitJSON(entries)
		}
		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, []string{e.Name, e.Description})
		}
		return output.Table([]string{"NAME", "DESCRIPTION"}, rows)
	},
}

type profileShowPayload struct {
	Description  string `json:"description,omitempty"`
	SourceDevice string `json:"source_device,omitempty"`
	*section.Snapshot
}

var profileShowCmd = &cobra.Command{
	Use:     "show <name>",
	Short:   "Show a saved profile",
	Example: "  tickerbox profile show home",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, description, sourceDevice, err := profile.LoadDescribed(args[0])
		if err != nil {
			return err
		}
		if jsonOut() {
			return output.EmitJSON(profileShowPayload{Description: description, SourceDevice: sourceDevice, Snapshot: s})
		}
		descriptionLine := description
		if descriptionLine == "" {
			descriptionLine = "(none)"
		}
		sourceLine := sourceDevice
		if sourceLine == "" {
			sourceLine = "-"
		}
		if err := output.KV([][2]string{{"description", descriptionLine}, {"source_device", sourceLine}}); err != nil {
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
		if !profileRmYes {
			ok, err := confirm(fmt.Sprintf("Remove profile %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := profile.Remove(args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed %s\n", args[0])
		return nil
	},
}

var profileApplyYes bool

var profileApplyCmd = &cobra.Command{
	Use:   "apply <name>",
	Short: "Apply a saved profile to the device",
	Long: "Applies only the sections present in the saved profile; sections it doesn't contain are left " +
		"untouched on the device.",
	Example: "  tickerbox profile apply home",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, sourceDevice, err := profile.LoadWithSource(args[0])
		if err != nil {
			return err
		}

		if sourceDevice != "" && resolvedDeviceName != "" && sourceDevice != resolvedDeviceName {
			if profileApplyYes {
				fmt.Fprintf(os.Stderr, "warning: applying profile %q (saved from device %q) to %q\n", args[0], sourceDevice, resolvedDeviceName)
			} else {
				ok, err := confirm(fmt.Sprintf("Profile %q was saved from device %q; the current target is %q. Apply anyway?", args[0], sourceDevice, resolvedDeviceName))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Println("aborted")
					return nil
				}
			}
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
		device, err := section.Capture(cmdContext(cmd), newClient(), include, true)
		if err != nil {
			return err
		}
		diffs := diff.Diff(device, prof, include, profileDiffShowSecrets)

		if jsonOut() {
			return output.EmitJSON(diffs)
		}
		return renderDiffTable(diffs, []string{"SECTION", "FIELD", "DEVICE", "PROFILE"}, "No differences", func(d diff.FieldDiff) []string {
			return []string{d.Section, d.Field, diff.FormatValue(d.Device), diff.FormatValue(d.Saved)}
		})
	},
}

func init() {
	profileSaveCmd.Flags().StringVar(&profileSaveInclude, "include", "", "comma-separated sections to capture (default "+strings.Join(section.DefaultInclude, ",")+")")
	profileSaveCmd.Flags().BoolVar(&profileSaveAll, "all", false, "also capture wifi and ap, including their secrets")
	profileSaveCmd.Flags().StringVarP(&profileSaveDescription, "description", "D", "", "optional human-readable description to store with the profile")

	profileRmCmd.Flags().BoolVarP(&profileRmYes, "yes", "y", false, "skip confirmation")

	profileApplyCmd.Flags().BoolVarP(&profileApplyYes, "yes", "y", false, "skip the source-device confirmation prompt")

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
