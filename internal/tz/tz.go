// SPDX-License-Identifier: MIT

package tz

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:generate go run gen/main.go

//go:embed timezones.json
var raw []byte

var (
	once sync.Once
	data map[string]string
)

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

func PosixFor(label string) (string, bool) {
	posix, ok := Load()[label]
	return posix, ok
}
