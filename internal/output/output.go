// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

func EmitJSON(v any) error {
	encoded, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}

func KV(pairs [][2]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for _, p := range pairs {
		_, _ = fmt.Fprintf(w, "%s:\t%s\n", p[0], p[1])
	}
	_ = w.Flush()
}

func Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			_, _ = fmt.Fprint(w, "\t")
		}
		_, _ = fmt.Fprint(w, h)
	}
	_, _ = fmt.Fprintln(w)
	for _, row := range rows {
		for i, c := range row {
			if i > 0 {
				_, _ = fmt.Fprint(w, "\t")
			}
			_, _ = fmt.Fprint(w, c)
		}
		_, _ = fmt.Fprintln(w)
	}
	_ = w.Flush()
}
