// SPDX-License-Identifier: MIT

package section

import (
	"fmt"
	"time"

	"github.com/openbunny/tickerbox-cli/internal/netfield"
	"github.com/openbunny/tickerbox-cli/internal/tz"
)

const (
	minBrightness = 10
	maxBrightness = 255

	minDisplayInterval = 10
	maxDisplayInterval = 60

	minAnimationSpeed = 10
	maxAnimationSpeed = 200

	maxSSIDLength = 32

	minAPPasswordLength = 8
	maxAPPasswordLength = 64

	minAPChannel = 1
	maxAPChannel = 14

	minAPClients = 1
	maxAPClients = 9
)

var iso8601Layouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func validateISO8601(field, value string) error {
	for _, layout := range iso8601Layouts {
		if _, err := time.Parse(layout, value); err == nil {
			return nil
		}
	}
	return fmt.Errorf("%s: invalid ISO8601 value %q", field, value)
}

func fieldInt(m map[string]any, key string) (int, bool, error) {
	v, ok := m[key]
	if !ok {
		return 0, false, nil
	}
	switch n := v.(type) {
	case float64:
		if n != float64(int(n)) {
			return 0, true, fmt.Errorf("%s must be a whole number, got %v", key, n)
		}
		return int(n), true, nil
	case int:
		return n, true, nil
	default:
		return 0, true, fmt.Errorf("%s must be a number, got %T", key, v)
	}
}

func fieldString(m map[string]any, key string) (string, bool, error) {
	v, ok := m[key]
	if !ok {
		return "", false, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", true, fmt.Errorf("%s must be a string, got %T", key, v)
	}
	return s, true, nil
}

func checkIntRange(m map[string]any, key string, min, max int) error {
	n, present, err := fieldInt(m, key)
	if err != nil {
		return err
	}
	if !present {
		return nil
	}
	if n < min || n > max {
		return fmt.Errorf("%s must be between %d and %d, got %d", key, min, max, n)
	}
	return nil
}

func checkSSID(m map[string]any) error {
	s, present, err := fieldString(m, "ssid")
	if err != nil {
		return err
	}
	if present && len(s) > maxSSIDLength {
		return fmt.Errorf("ssid exceeds %d characters", maxSSIDLength)
	}
	return nil
}

func checkIP(m map[string]any, key string) error {
	s, present, err := fieldString(m, key)
	if err != nil {
		return err
	}
	if !present || s == "" {
		return nil
	}
	return netfield.IP(key, s)
}

func checkSubnetMask(m map[string]any) error {
	s, present, err := fieldString(m, "subnet_mask")
	if err != nil {
		return err
	}
	if !present || s == "" {
		return nil
	}
	return netfield.Netmask("subnet_mask", s)
}

func checkTZLabel(m map[string]any) error {
	s, present, err := fieldString(m, "tz_label")
	if err != nil {
		return err
	}
	if !present || s == "" {
		return nil
	}
	if _, ok := tz.PosixFor(s); !ok {
		return fmt.Errorf("tz_label must be a known timezone label, got %q", s)
	}
	return nil
}

func validateDisplay(m map[string]any) error {
	if err := checkIntRange(m, "brightness", minBrightness, maxBrightness); err != nil {
		return err
	}
	if err := checkIntRange(m, "changeInterval", minDisplayInterval, maxDisplayInterval); err != nil {
		return err
	}
	if s, present, err := fieldString(m, "sleepStart"); err != nil {
		return err
	} else if present && s != "" {
		if err := validateISO8601("sleepStart", s); err != nil {
			return err
		}
	}
	if s, present, err := fieldString(m, "sleepEnd"); err != nil {
		return err
	} else if present && s != "" {
		if err := validateISO8601("sleepEnd", s); err != nil {
			return err
		}
	}
	return nil
}

func validateClock(m map[string]any) error {
	if err := checkIntRange(m, "animationSpeed", minAnimationSpeed, maxAnimationSpeed); err != nil {
		return err
	}
	return checkTZLabel(m)
}

func validateNTP(m map[string]any) error {
	return checkTZLabel(m)
}

func validateWifi(m map[string]any) error {
	if err := checkSSID(m); err != nil {
		return err
	}
	for _, key := range []string{"local_ip", "gateway_ip", "dns_ip_1", "dns_ip_2"} {
		if err := checkIP(m, key); err != nil {
			return err
		}
	}
	return checkSubnetMask(m)
}

func validateAP(m map[string]any) error {
	if err := checkSSID(m); err != nil {
		return err
	}
	if s, present, err := fieldString(m, "password"); err != nil {
		return err
	} else if present && s != "" {
		if len(s) < minAPPasswordLength || len(s) > maxAPPasswordLength {
			return fmt.Errorf("password must be %d-%d characters, got %d", minAPPasswordLength, maxAPPasswordLength, len(s))
		}
	}
	if err := checkIntRange(m, "channel", minAPChannel, maxAPChannel); err != nil {
		return err
	}
	if err := checkIntRange(m, "max_clients", minAPClients, maxAPClients); err != nil {
		return err
	}
	for _, key := range []string{"local_ip", "gateway_ip"} {
		if err := checkIP(m, key); err != nil {
			return err
		}
	}
	return checkSubnetMask(m)
}

func validateSectionMap(name string, m map[string]any) error {
	var err error
	switch name {
	case SectionDisplay:
		err = validateDisplay(m)
	case SectionClock:
		err = validateClock(m)
	case SectionNTP:
		err = validateNTP(m)
	case SectionWifi:
		err = validateWifi(m)
	case SectionAP:
		err = validateAP(m)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
