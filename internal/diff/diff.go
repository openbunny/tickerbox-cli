// SPDX-License-Identifier: MIT

package diff

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

type FieldDiff struct {
	Section string `json:"section"`
	Field   string `json:"field"`
	Device  any    `json:"device"`
	Saved   any    `json:"saved"`
}

// Diff compares device against saved section by section. A key present on both
// sides with a different value always produces a row, masked when it is a secret
// field and showSecrets is false; comparing the real values first and masking only
// the displayed result is what lets an unchanged secret produce zero rows, unlike
// masking (or stripping) before comparison.
func Diff(device, saved *section.Snapshot, sections []string, showSecrets bool) []FieldDiff {
	var diffs []FieldDiff
	for _, name := range sections {
		if name == section.SectionTickers {
			diffs = append(diffs, diffTickers(device.Tickers, saved.Tickers)...)
			continue
		}
		diffs = append(diffs, diffMap(name, sectionMap(device, name), sectionMap(saved, name), showSecrets)...)
	}
	return diffs
}

func sectionMap(s *section.Snapshot, name string) map[string]any {
	switch name {
	case section.SectionDisplay:
		return s.Display
	case section.SectionClock:
		return s.Clock
	case section.SectionNTP:
		return s.NTP
	case section.SectionWifi:
		return s.Wifi
	case section.SectionAP:
		return s.AP
	default:
		return nil
	}
}

func diffMap(name string, device, saved map[string]any, showSecrets bool) []FieldDiff {
	keys := make(map[string]struct{}, len(device)+len(saved))
	for k := range device {
		keys[k] = struct{}{}
	}
	for k := range saved {
		keys[k] = struct{}{}
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var diffs []FieldDiff
	for _, k := range sorted {
		dv, dok := device[k]
		sv, sok := saved[k]
		if dok && sok && reflect.DeepEqual(dv, sv) {
			continue
		}
		dispD, dispS := fieldValue(dv, dok), fieldValue(sv, sok)
		if !showSecrets && section.IsSecretField(k) {
			dispD, dispS = maskValue(dispD), maskValue(dispS)
		}
		diffs = append(diffs, FieldDiff{Section: name, Field: k, Device: dispD, Saved: dispS})
	}
	return diffs
}

func fieldValue(v any, present bool) any {
	if !present {
		return nil
	}
	return v
}

func maskValue(v any) any {
	if v == nil {
		return nil
	}
	return "********"
}

func diffTickers(device, saved []tickers.Entry) []FieldDiff {
	n := max(len(device), len(saved))

	var diffs []FieldDiff
	for i := 0; i < n; i++ {
		dOK := i < len(device)
		sOK := i < len(saved)
		var d, s tickers.Entry
		if dOK {
			d = device[i]
		}
		if sOK {
			s = saved[i]
		}
		prefix := fmt.Sprintf("tickers[%d].", i)
		diffs = append(diffs, diffTickerField(prefix+"type", d.Type, s.Type, dOK, sOK)...)
		diffs = append(diffs, diffTickerField(prefix+"ticker", d.Ticker, s.Ticker, dOK, sOK)...)
		diffs = append(diffs, diffTickerField(prefix+"time", d.Time, s.Time, dOK, sOK)...)
		diffs = append(diffs, diffTickerField(prefix+"currency", d.Currency, s.Currency, dOK, sOK)...)
	}
	return diffs
}

func diffTickerField(field, device, saved string, dOK, sOK bool) []FieldDiff {
	if dOK && sOK && device == saved {
		return nil
	}
	if !dOK && !sOK {
		return nil
	}
	return []FieldDiff{{Section: section.SectionTickers, Field: field, Device: fieldValue(device, dOK), Saved: fieldValue(saved, sOK)}}
}

func FormatValue(v any) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprint(v)
}
