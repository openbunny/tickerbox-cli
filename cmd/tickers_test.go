// SPDX-License-Identifier: MIT

package cmd

import "testing"

func TestIsIndex(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "digits", in: "42", want: true},
		{name: "zero", in: "0", want: true},
		{name: "empty", in: "", want: false},
		{name: "ticker symbol", in: "BTC", want: false},
		{name: "mixed digits and letters", in: "4a", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIndex(tt.in); got != tt.want {
				t.Errorf("isIndex(%q) = %v; want %v", tt.in, got, tt.want)
			}
		})
	}
}
