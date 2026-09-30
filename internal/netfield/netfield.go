// SPDX-License-Identifier: MIT

package netfield

import (
	"fmt"
	"net"
)

func IP(field, value string) error {
	if net.ParseIP(value) == nil {
		return fmt.Errorf("invalid %s %q: not an IP address", field, value)
	}
	return nil
}

func Netmask(field, value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("invalid %s %q: not an IP address", field, value)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return fmt.Errorf("invalid %s %q: not an IPv4 subnet mask", field, value)
	}
	if _, bits := net.IPv4Mask(ip4[0], ip4[1], ip4[2], ip4[3]).Size(); bits == 0 {
		return fmt.Errorf("invalid %s %q: not a contiguous subnet mask", field, value)
	}
	return nil
}
