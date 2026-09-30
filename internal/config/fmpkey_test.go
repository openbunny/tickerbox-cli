// SPDX-License-Identifier: MIT

package config

import "testing"

func TestResolveFMPAPIKeyNeitherSet(t *testing.T) {
	t.Setenv(EnvFMPAPIKey, "")
	c := &Config{}
	if _, ok := c.ResolveFMPAPIKey(); ok {
		t.Error("ResolveFMPAPIKey() ok = true, want false with no env and no stored key")
	}
}

func TestResolveFMPAPIKeyFromConfig(t *testing.T) {
	t.Setenv(EnvFMPAPIKey, "")
	c := &Config{FMPAPIKey: "stored-key"}
	key, ok := c.ResolveFMPAPIKey()
	if !ok || key != "stored-key" {
		t.Errorf("ResolveFMPAPIKey() = %q, %v; want %q, true", key, ok, "stored-key")
	}
}

func TestResolveFMPAPIKeyFromEnv(t *testing.T) {
	t.Setenv(EnvFMPAPIKey, "env-key")
	c := &Config{FMPAPIKey: "stored-key"}
	key, ok := c.ResolveFMPAPIKey()
	if !ok || key != "env-key" {
		t.Errorf("ResolveFMPAPIKey() = %q, %v; want %q, true", key, ok, "env-key")
	}
}

func TestResolveFMPAPIKeyEnvOverridesStoredValue(t *testing.T) {
	t.Setenv(EnvFMPAPIKey, "env-key")
	c := &Config{FMPAPIKey: "stored-key"}
	key, _ := c.ResolveFMPAPIKey()
	if key == "stored-key" {
		t.Error("ResolveFMPAPIKey() returned the stored key despite the env var being set")
	}
}

func TestSetFMPAPIKeyRejectsEmpty(t *testing.T) {
	c := &Config{}
	if err := c.SetFMPAPIKey(""); err == nil {
		t.Error("SetFMPAPIKey(\"\") error = nil, want error")
	}
}

func TestSetFMPAPIKeyStores(t *testing.T) {
	c := &Config{}
	if err := c.SetFMPAPIKey("abc123"); err != nil {
		t.Fatalf("SetFMPAPIKey() error = %v", err)
	}
	if c.FMPAPIKey != "abc123" {
		t.Errorf("FMPAPIKey = %q, want %q", c.FMPAPIKey, "abc123")
	}
}
