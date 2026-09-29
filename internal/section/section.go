// SPDX-License-Identifier: MIT

package section

import (
	"fmt"
	"strings"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

const (
	SectionTickers = "tickers"
	SectionDisplay = "display"
	SectionClock   = "clock"
	SectionNTP     = "ntp"
	SectionWifi    = "wifi"
	SectionAP      = "ap"
)

var Sections = []string{SectionTickers, SectionDisplay, SectionClock, SectionNTP, SectionWifi, SectionAP}

var DefaultInclude = []string{SectionTickers, SectionDisplay, SectionClock, SectionNTP}

const (
	pathCoinSetupState  = "coinSetupState"
	pathSettingsState   = "settingsState"
	pathClockSetupState = "clockSetupState"
	pathNTPSettings     = "ntpSettings"
	pathWifiSettings    = "wifiSettings"
	pathAPSettings      = "apSettings"
)

var mapSectionPath = map[string]string{
	SectionDisplay: pathSettingsState,
	SectionClock:   pathClockSetupState,
	SectionNTP:     pathNTPSettings,
	SectionWifi:    pathWifiSettings,
	SectionAP:      pathAPSettings,
}

var secretSections = map[string]bool{SectionWifi: true, SectionAP: true}

type Snapshot struct {
	Wifi    map[string]any  `json:"wifi,omitempty"`
	AP      map[string]any  `json:"ap,omitempty"`
	NTP     map[string]any  `json:"ntp,omitempty"`
	Display map[string]any  `json:"display,omitempty"`
	Clock   map[string]any  `json:"clock,omitempty"`
	Tickers []tickers.Entry `json:"tickers,omitempty"`
}

func Capture(c *client.Client, include []string, withSecrets bool) (*Snapshot, error) {
	s := &Snapshot{}
	for _, name := range include {
		if name == SectionTickers {
			entries, err := getTickers(c)
			if err != nil {
				return nil, err
			}
			s.Tickers = entries
			continue
		}
		path, ok := mapSectionPath[name]
		if !ok {
			return nil, fmt.Errorf("unknown section %q", name)
		}
		m, err := getMap(c, path)
		if err != nil {
			return nil, err
		}
		if secretSections[name] && !withSecrets {
			m = stripSecrets(m)
		}
		setSection(s, name, m)
	}
	return s, nil
}

func Apply(c *client.Client, s *Snapshot, include []string) error {
	for _, name := range include {
		if name == SectionTickers {
			if err := postTickers(c, s.Tickers); err != nil {
				return err
			}
			continue
		}
		path, ok := mapSectionPath[name]
		if !ok {
			return fmt.Errorf("unknown section %q", name)
		}
		if err := postMap(c, path, getSection(s, name)); err != nil {
			return err
		}
	}
	return nil
}

func setSection(s *Snapshot, name string, m map[string]any) {
	switch name {
	case SectionDisplay:
		s.Display = m
	case SectionClock:
		s.Clock = m
	case SectionNTP:
		s.NTP = m
	case SectionWifi:
		s.Wifi = m
	case SectionAP:
		s.AP = m
	}
}

func getSection(s *Snapshot, name string) map[string]any {
	switch name {
	case SectionDisplay:
		return s.Display
	case SectionClock:
		return s.Clock
	case SectionNTP:
		return s.NTP
	case SectionWifi:
		return s.Wifi
	case SectionAP:
		return s.AP
	default:
		return nil
	}
}

func getMap(c *client.Client, path string) (map[string]any, error) {
	var m map[string]any
	if err := c.Get(path, &m); err != nil {
		return nil, fmt.Errorf("get %s: %w", path, err)
	}
	return m, nil
}

func postMap(c *client.Client, path string, m map[string]any) error {
	if err := c.Post(path, m); err != nil {
		return fmt.Errorf("post %s: %w", path, err)
	}
	return nil
}

func stripSecrets(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "password") || strings.Contains(lower, "secret") {
			continue
		}
		out[k] = v
	}
	return out
}

func getTickers(c *client.Client) ([]tickers.Entry, error) {
	var state tickers.State
	if err := c.Get(pathCoinSetupState, &state); err != nil {
		return nil, fmt.Errorf("get %s: %w", pathCoinSetupState, err)
	}
	entries, err := tickers.Decode(state)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", pathCoinSetupState, err)
	}
	return entries, nil
}

func postTickers(c *client.Client, entries []tickers.Entry) error {
	if err := c.Post(pathCoinSetupState, tickers.Encode(entries)); err != nil {
		return fmt.Errorf("post %s: %w", pathCoinSetupState, err)
	}
	return nil
}
