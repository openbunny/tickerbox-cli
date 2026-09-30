// SPDX-License-Identifier: MIT

package section

import (
	"context"
	"fmt"
	"strings"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/concurrent"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

const captureConcurrency = 3

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

type captureResult struct {
	name    string
	tickers []tickers.Entry
	m       map[string]any
	err     error
}

func Capture(ctx context.Context, c *client.Client, include []string, withSecrets bool) (*Snapshot, error) {
	results := make([]captureResult, len(include))
	tasks := make([]func(), len(include))
	for i, name := range include {
		i, name := i, name
		tasks[i] = func() {
			if name == SectionTickers {
				entries, err := getTickers(ctx, c)
				results[i] = captureResult{name: name, tickers: entries, err: err}
				return
			}
			path, ok := mapSectionPath[name]
			if !ok {
				results[i] = captureResult{name: name, err: fmt.Errorf("unknown section %q", name)}
				return
			}
			m, err := getMap(ctx, c, path)
			results[i] = captureResult{name: name, m: m, err: err}
		}
	}
	concurrent.Run(captureConcurrency, tasks...)

	s := &Snapshot{}
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		if r.name == SectionTickers {
			s.Tickers = r.tickers
			continue
		}
		m := r.m
		if secretSections[r.name] && !withSecrets {
			m = stripSecrets(m)
		}
		setSection(s, r.name, m)
	}
	return s, nil
}

func Apply(ctx context.Context, c *client.Client, s *Snapshot, include []string) error {
	for _, name := range include {
		if name == SectionTickers {
			if err := postTickers(ctx, c, s.Tickers); err != nil {
				return err
			}
			continue
		}
		path, ok := mapSectionPath[name]
		if !ok {
			return fmt.Errorf("unknown section %q", name)
		}
		if err := postMap(ctx, c, path, getSection(s, name)); err != nil {
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

func getMap(ctx context.Context, c *client.Client, path string) (map[string]any, error) {
	var m map[string]any
	if err := c.Get(ctx, path, &m); err != nil {
		return nil, fmt.Errorf("get %s: %w", path, err)
	}
	return m, nil
}

func postMap(ctx context.Context, c *client.Client, path string, m map[string]any) error {
	if err := c.Post(ctx, path, m); err != nil {
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

func getTickers(ctx context.Context, c *client.Client) ([]tickers.Entry, error) {
	var state tickers.State
	if err := c.Get(ctx, pathCoinSetupState, &state); err != nil {
		return nil, fmt.Errorf("get %s: %w", pathCoinSetupState, err)
	}
	entries, err := tickers.Decode(state)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", pathCoinSetupState, err)
	}
	return entries, nil
}

func postTickers(ctx context.Context, c *client.Client, entries []tickers.Entry) error {
	if err := c.Post(ctx, pathCoinSetupState, tickers.Encode(entries)); err != nil {
		return fmt.Errorf("post %s: %w", pathCoinSetupState, err)
	}
	return nil
}
