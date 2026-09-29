package tz

import "testing"

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
