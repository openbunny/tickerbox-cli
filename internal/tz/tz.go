// Package tz holds the CLI's built-in table of timezone labels and their
// POSIX TZ strings, embedded at build time so no network lookup is needed.
package tz

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed timezones.json
var raw []byte

var (
	once sync.Once
	data map[string]string
)

// Load returns the built-in timezone table, keyed by label, parsing the
// embedded JSON on first call and panicking if it is malformed (a build-time
// invariant, never a runtime input).
func Load() map[string]string {
	once.Do(func() {
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			panic(fmt.Errorf("parse embedded timezones.json: %w", err))
		}
		data = m
	})
	return data
}

// PosixFor returns the POSIX TZ string for label and whether label is known.
func PosixFor(label string) (string, bool) {
	posix, ok := Load()[label]
	return posix, ok
}
