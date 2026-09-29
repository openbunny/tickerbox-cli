// Package output renders command results to stdout, as indented JSON or as
// a tab-aligned key/value list or table.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

// EmitJSON prints v to stdout as indented JSON followed by a newline. It
// returns an error if v cannot be marshaled.
func EmitJSON(v any) error {
	encoded, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}

// KV prints pairs to stdout as tab-aligned "key: value" lines, in order.
func KV(pairs [][2]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for _, p := range pairs {
		_, _ = fmt.Fprintf(w, "%s:\t%s\n", p[0], p[1])
	}
	_ = w.Flush()
}

// Table prints headers and rows to stdout as a tab-aligned table. Each row
// is truncated or padded implicitly by the tabwriter; callers are
// responsible for giving every row the same number of columns as headers.
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
