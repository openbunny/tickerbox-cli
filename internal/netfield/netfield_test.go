// SPDX-License-Identifier: MIT

package netfield

import "testing"

func TestIP(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid IPv4", value: "192.168.1.50"},
		{name: "valid IPv6", value: "::1"},
		{name: "empty", value: "", wantErr: true},
		{name: "not an IP", value: "192.168.1.1x", wantErr: true},
		{name: "hostname", value: "tickerbox.local", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IP("--local-ip", tt.value)
			if tt.wantErr && err == nil {
				t.Fatalf("IP(%q) = nil error; want error", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("IP(%q) = %v; want nil", tt.value, err)
			}
		})
	}
}

func TestNetmask(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid /24", value: "255.255.255.0"},
		{name: "valid /16", value: "255.255.0.0"},
		{name: "not an IP", value: "not-a-mask", wantErr: true},
		{name: "non-contiguous bits", value: "255.0.255.0", wantErr: true},
		{name: "IPv6 address", value: "::1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Netmask("--subnet-mask", tt.value)
			if tt.wantErr && err == nil {
				t.Fatalf("Netmask(%q) = nil error; want error", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Netmask(%q) = %v; want nil", tt.value, err)
			}
		})
	}
}
