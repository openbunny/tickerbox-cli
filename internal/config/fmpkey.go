// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"os"
)

const EnvFMPAPIKey = "TICKERBOX_FMP_API_KEY"

// ResolveFMPAPIKey returns the Financial Modeling Prep API key from
// $TICKERBOX_FMP_API_KEY, falling back to the key persisted in config.toml by
// `tickerbox fmp set-key`. ok is false when neither supplies one.
func (c *Config) ResolveFMPAPIKey() (key string, ok bool) {
	if v := os.Getenv(EnvFMPAPIKey); v != "" {
		return v, true
	}
	if c.FMPAPIKey != "" {
		return c.FMPAPIKey, true
	}
	return "", false
}

func (c *Config) SetFMPAPIKey(key string) error {
	if key == "" {
		return errors.New("API key must not be empty")
	}
	c.FMPAPIKey = key
	return nil
}
