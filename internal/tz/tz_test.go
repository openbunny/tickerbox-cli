// SPDX-License-Identifier: MIT

package tz

import (
	"regexp"
	"testing"
)

var (
	ianaLabelPattern = regexp.MustCompile(`^[A-Za-z_+-]+(?:/[A-Za-z0-9_+-]+)+$`)
	posixTZPattern   = regexp.MustCompile(`^` +
		`(?:[A-Za-z]+|<[+\-0-9A-Za-z]+>)[+-]?\d{1,3}(?::\d{2}){0,2}` +
		`(?:(?:[A-Za-z]+|<[+\-0-9A-Za-z]+>)(?:[+-]?\d{1,3}(?::\d{2}){0,2})?` +
		`(?:,(?:J\d{1,3}|\d{1,3}|M\d{1,2}\.\d\.\d)(?:/[+-]?\d{1,3}(?::\d{2}){0,2})?` +
		`,(?:J\d{1,3}|\d{1,3}|M\d{1,2}\.\d\.\d)(?:/[+-]?\d{1,3}(?::\d{2}){0,2})?)?)?$`)
)

func TestPosixTZPattern(t *testing.T) {
	tests := []struct {
		name  string
		posix string
		want  bool
	}{
		{name: "fixed offset", posix: "EET-2", want: true},
		{name: "dst with default offset and rule", posix: "CET-1CEST,M3.5.0,M10.5.0/3", want: true},
		{name: "half hour offset", posix: "ACST-9:30ACDT,M10.1.0,M4.1.0/3", want: true},
		{name: "bracketed numeric abbreviation", posix: "<+14>-14", want: true},
		{name: "julian day rule", posix: "UNK-3:30UNK,J79/24,J263/24", want: true},
		{name: "negative rule time", posix: "UNK3UNK,M3.5.0/-2,M10.5.0/-1", want: true},
		{name: "empty string", posix: "", want: false},
		{name: "no offset", posix: "EET", want: false},
		{name: "trailing garbage", posix: "EET-2;rm -rf", want: false},
		{name: "lone comma", posix: "EET-2,", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := posixTZPattern.MatchString(tt.posix); got != tt.want {
				t.Errorf("posixTZPattern.MatchString(%q) = %v; want %v", tt.posix, got, tt.want)
			}
		})
	}
}

func TestLoadEntries(t *testing.T) {
	entries := Load()
	if len(entries) == 0 {
		t.Fatal("Load() returned no entries")
	}
	for label, posix := range entries {
		t.Run(label, func(t *testing.T) {
			if !ianaLabelPattern.MatchString(label) {
				t.Errorf("label %q is not a slash-separated IANA label", label)
			}
			if !posixTZPattern.MatchString(posix) {
				t.Errorf("posix %q for label %q is not a plausible POSIX TZ string", posix, label)
			}
		})
	}
}

func TestPosixFor(t *testing.T) {
	tests := []struct {
		name      string
		label     string
		wantPosix string
		wantOK    bool
	}{
		{name: "known label", label: "Africa/Cairo", wantPosix: "EET-2", wantOK: true},
		{name: "unknown label", label: "Moon/Base", wantOK: false},
		{name: "empty label", label: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posix, ok := PosixFor(tt.label)
			if ok != tt.wantOK {
				t.Fatalf("got ok=%v; want %v", ok, tt.wantOK)
			}
			if ok && posix != tt.wantPosix {
				t.Errorf("got posix %q; want %q", posix, tt.wantPosix)
			}
		})
	}
}
