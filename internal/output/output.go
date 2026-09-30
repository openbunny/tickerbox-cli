// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

const (
	tabMinWidth = 0
	tabWidth    = 4
	tabPadding  = 2
	tabPadChar  = ' '
	tabFlags    = 0
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

func KV(pairs [][2]string) error {
	w := tabwriter.NewWriter(os.Stdout, tabMinWidth, tabWidth, tabPadding, tabPadChar, tabFlags)
	for _, p := range pairs {
		if _, err := fmt.Fprintf(w, "%s:\t%s\n", p[0], p[1]); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	return nil
}

func Table(headers []string, rows [][]string) error {
	w := tabwriter.NewWriter(os.Stdout, tabMinWidth, tabWidth, tabPadding, tabPadChar, tabFlags)
	for i, h := range headers {
		if i > 0 {
			if _, err := fmt.Fprint(w, "\t"); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}
		if _, err := fmt.Fprint(w, h); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, row := range rows {
		for i, c := range row {
			if i > 0 {
				if _, err := fmt.Fprint(w, "\t"); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
			}
			if _, err := fmt.Fprint(w, c); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	return nil
}
