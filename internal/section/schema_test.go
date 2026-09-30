// SPDX-License-Identifier: MIT

package section

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

type jsonSchema struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Defs       map[string]jsonSchema      `json:"$defs"`
}

func loadSnapshotSchema(t *testing.T) jsonSchema {
	t.Helper()
	data, err := os.ReadFile("../../docs/schema/snapshot.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema jsonSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	return schema
}

func propertyNames(m map[string]json.RawMessage) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func jsonFieldNames(t *testing.T, v any) []string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %T: %v", v, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal %T: %v", v, err)
	}
	return propertyNames(fields)
}

func assertSameNames(t *testing.T, label string, schemaNames, structNames []string) {
	t.Helper()
	if len(schemaNames) != len(structNames) {
		t.Errorf("%s: schema properties %v != struct JSON fields %v", label, schemaNames, structNames)
		return
	}
	for i := range schemaNames {
		if schemaNames[i] != structNames[i] {
			t.Errorf("%s: schema properties %v != struct JSON fields %v", label, schemaNames, structNames)
			return
		}
	}
}

func TestSnapshotSchemaMatchesStruct(t *testing.T) {
	schema := loadSnapshotSchema(t)

	snap := Snapshot{
		Wifi:    map[string]any{"ssid": "home"},
		AP:      map[string]any{"ssid": "box-ap"},
		NTP:     map[string]any{"server": "pool.ntp.org"},
		Display: map[string]any{"brightness": 200},
		Clock:   map[string]any{"enabled": true},
		Tickers: []tickers.Entry{
			{Type: tickers.TypeCrypto, Ticker: "BTC", Time: tickers.Time1Min, Currency: tickers.CurrencyUSD},
		},
	}
	assertSameNames(t, "Snapshot", propertyNames(schema.Properties), jsonFieldNames(t, snap))

	entryDef, ok := schema.Defs["tickerEntry"]
	if !ok {
		t.Fatal(`schema $defs["tickerEntry"] not found`)
	}
	assertSameNames(t, "tickers.Entry", propertyNames(entryDef.Properties), jsonFieldNames(t, snap.Tickers[0]))
}
