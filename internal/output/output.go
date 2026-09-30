// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"strings"
	"text/tabwriter"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

const (
	tabMinWidth = 0
	tabWidth    = 4
	tabPadding  = 2
	tabPadChar  = ' '
	tabFlags    = 0

	tableColumnGap = 2
)

var (
	tableHeaderStyle = lipgloss.NewStyle().Bold(true).PaddingRight(tableColumnGap)
	tableCellStyle   = lipgloss.NewStyle().PaddingRight(tableColumnGap)
	kvKeyStyle       = lipgloss.NewStyle().Bold(true)
)

func EmitJSON(v any) error {
	encoded, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	if _, err := fmt.Println(string(encoded)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func DeviceBanner(name, host string) error {
	if _, err := fmt.Fprintf(os.Stdout, "device: %s (%s)\n", name, host); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func KV(pairs [][2]string) error {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, tabMinWidth, tabWidth, tabPadding, tabPadChar, tabFlags)
	for _, p := range pairs {
		if _, err := fmt.Fprintf(w, "%s:\t%s\n", kvKeyStyle.Render(p[0]), p[1]); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	if _, err := lipgloss.Fprint(os.Stdout, buf.String()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// ColumnColor picks the foreground color for a cell's value in one table
// column, letting a caller highlight that column by value.
type ColumnColor func(value string) color.Color

func Table(headers []string, rows [][]string) error {
	return renderTable(headers, rows, -1, nil)
}

// ColoredTable renders headers and rows like Table, additionally coloring
// every cell in column by colorFn(cell value).
func ColoredTable(headers []string, rows [][]string, column int, colorFn ColumnColor) error {
	return renderTable(headers, rows, column, colorFn)
}

func renderTable(headers []string, rows [][]string, colorColumn int, colorFn ColumnColor) error {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderHeader(false).BorderColumn(false).BorderRow(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return tableHeaderStyle
			case col == colorColumn && colorFn != nil:
				return tableCellStyle.Foreground(colorFn(rows[row][col]))
			default:
				return tableCellStyle
			}
		})

	lines := strings.Split(t.Render(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}

	if _, err := lipgloss.Fprintln(os.Stdout, strings.Join(lines, "\n")); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
