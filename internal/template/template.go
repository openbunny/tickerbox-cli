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

type Preset struct {
	Description string          `json:"description"`
	Entries     []tickers.Entry `json:"entries"`
}

var presets = mustLoadPresets()

func mustLoadPresets() map[string]Preset {
	var m map[string]Preset
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
	p, ok := presets[name]
	if !ok {
		return nil, false
	}
	out := make([]tickers.Entry, len(p.Entries))
	copy(out, p.Entries)
	return out, true
}

func Describe(name string) (string, bool) {
	p, ok := presets[name]
	if !ok {
		return "", false
	}
	return p.Description, true
}
