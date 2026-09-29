package cmd

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/tz"
)

var tzListGrep string

var tzCmd = &cobra.Command{
	Use:   "tz",
	Short: "Time zones",
}

var tzListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known time zone labels and their POSIX strings",
	RunE: func(cmd *cobra.Command, args []string) error {
		filtered := map[string]string{}
		needle := strings.ToLower(tzListGrep)
		for label, posix := range tz.Load() {
			if needle == "" || strings.Contains(strings.ToLower(label), needle) {
				filtered[label] = posix
			}
		}

		if jsonOut() {
			return output.EmitJSON(filtered)
		}

		labels := make([]string, 0, len(filtered))
		for label := range filtered {
			labels = append(labels, label)
		}
		sort.Strings(labels)

		rows := make([][]string, 0, len(labels))
		for _, label := range labels {
			rows = append(rows, []string{label, filtered[label]})
		}
		output.Table([]string{"LABEL", "POSIX"}, rows)
		return nil
	},
}

func init() {
	tzListCmd.Flags().StringVar(&tzListGrep, "grep", "", "case-insensitive filter on label")
	tzCmd.AddCommand(tzListCmd)
	rootCmd.AddCommand(tzCmd)
}
