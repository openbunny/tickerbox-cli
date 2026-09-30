// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/diff"
	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/section"
)

const configSnapshotFilePerm = 0o600

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Device configuration snapshot: export, import, diff",
	Long: "Raw export, import, and diff of one device's live config snapshot as a JSON file. Distinct from " +
		"`tickerbox profile`, which stores named, reusable bundles rather than a one-off file, and unrelated " +
		"to `tickerbox device`, which manages which device this CLI targets.",
}

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

func configSectionPresent(s *section.Snapshot, name string) bool {
	if name == section.SectionTickers {
		return s.Tickers != nil
	}
	_, ok := snapshotMapField(s, name)
	return ok
}

func configPresentSections(s *section.Snapshot) []string {
	var names []string
	for _, name := range section.Sections {
		if configSectionPresent(s, name) {
			names = append(names, name)
		}
	}
	return names
}

var (
	configExportFile        string
	configExportShowSecrets bool
)

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Capture every device config section as a JSON snapshot",
	Long: "Captures every config section, including wifi and ap. --show-secrets includes their passwords; " +
		"otherwise they're omitted from the file.",
	Example: "  tickerbox config export --file backup.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := section.Capture(cmdContext(cmd), newClient(), section.Sections, configExportShowSecrets)
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
	Use:     "import <file>",
	Short:   "Apply a saved config snapshot to the device",
	Long:    "Applies only the sections present in the file. Prompts for confirmation unless --yes.",
	Example: "  tickerbox config import backup.json --yes",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := loadSnapshot(args[0])
		if err != nil {
			return err
		}

		include := configPresentSections(snap)
		if len(include) == 0 {
			return fmt.Errorf("%s carries no config sections", args[0])
		}

		if !configImportYes {
			ok, err := confirm(fmt.Sprintf("Apply %s from %s to the device?", strings.Join(include, ", "), args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}

		if err := section.Apply(cmdContext(cmd), newClient(), snap, include); err != nil {
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
	Long: "Compares every section. A differing wifi/ap secret field is shown masked as ******** by default; " +
		"--show-secrets compares and shows their real values instead.",
	Example: "  tickerbox config diff backup.json",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		saved, err := loadSnapshot(args[0])
		if err != nil {
			return err
		}

		device, err := section.Capture(cmdContext(cmd), newClient(), section.Sections, true)
		if err != nil {
			return fmt.Errorf("capture device config: %w", err)
		}

		diffs := diff.Diff(device, saved, section.Sections, configDiffShowSecrets)
		if jsonOut() {
			return output.EmitJSON(diffs)
		}
		return renderDiffTable(diffs, []string{"SECTION", "FIELD", "DEVICE", "SAVED"}, "no differences", func(d diff.FieldDiff) []string {
			return []string{d.Section, d.Field, diff.FormatValue(d.Device), diff.FormatValue(d.Saved)}
		})
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
