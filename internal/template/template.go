// SPDX-License-Identifier: MIT

package template

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

//go:embed presets.json
var presetsJSON []byte

var presets = mustLoadPresets()

func mustLoadPresets() map[string][]tickers.Entry {
	var m map[string][]tickers.Entry
	if err := json.Unmarshal(presetsJSON, &m); err != nil {
		panic(fmt.Sprintf("template: decode presets.json: %v", err))
	}
	return m
}

func List() []string {
	names := make([]string, 0, len(presets))
	for name := range presets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Get(name string) ([]tickers.Entry, bool) {
	entries, ok := presets[name]
	if !ok {
		return nil, false
	}
	out := make([]tickers.Entry, len(entries))
	copy(out, entries)
	return out, true
}
