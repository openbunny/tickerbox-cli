// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/openbunny/tickerbox-cli/internal/output"
)

func renderDiffTable[T any](diffs []T, headers []string, emptyMessage string, row func(T) []string) error {
	if len(diffs) == 0 {
		fmt.Println(emptyMessage)
		return nil
	}
	rows := make([][]string, len(diffs))
	for i, d := range diffs {
		rows[i] = row(d)
	}
	return output.Table(headers, rows)
}
