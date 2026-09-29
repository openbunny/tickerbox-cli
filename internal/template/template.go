// Package template holds the CLI's built-in, read-only ticker list presets.
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

// mustLoadPresets decodes presets.json, embedded in the binary at build
// time: a decode failure here is a build defect, not a runtime condition a
// caller can act on, so it panics rather than threading an error through
// every caller of List and Get.
func mustLoadPresets() map[string][]tickers.Entry {
	var m map[string][]tickers.Entry
	if err := json.Unmarshal(presetsJSON, &m); err != nil {
		panic(fmt.Sprintf("template: decode presets.json: %v", err))
	}
	return m
}

// List returns the names of all built-in presets, sorted.
func List() []string {
	names := make([]string, 0, len(presets))
	for name := range presets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Get returns the named preset's entries and whether that preset exists.
// The returned slice is a copy: callers may modify it without affecting the
// built-in preset.
func Get(name string) ([]tickers.Entry, bool) {
	entries, ok := presets[name]
	if !ok {
		return nil, false
	}
	out := make([]tickers.Entry, len(entries))
	copy(out, entries)
	return out, true
}
