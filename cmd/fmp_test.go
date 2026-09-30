// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/config"
)

func TestFmpSetKeyPersistsAndMasksOutput(t *testing.T) {
	useTempDeviceConfigDir(t)
	t.Setenv(config.EnvFMPAPIKey, "")

	stdout := captureStdout(t, func() {
		if err := fmpSetKeyCmd.RunE(fmpSetKeyCmd, []string{"super-secret-key"}); err != nil {
			t.Fatalf("fmpSetKeyCmd.RunE() = %v", err)
		}
	})

	if strings.Contains(stdout, "super-secret-key") {
		t.Errorf("stdout = %q; the raw key must not appear", stdout)
	}
	if !strings.Contains(stdout, "********") {
		t.Errorf("stdout = %q; want it to contain the masked key", stdout)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.FMPAPIKey != "super-secret-key" {
		t.Errorf("FMPAPIKey = %q, want %q", cfg.FMPAPIKey, "super-secret-key")
	}

	key, ok := cfg.ResolveFMPAPIKey()
	if !ok || key != "super-secret-key" {
		t.Errorf("ResolveFMPAPIKey() = %q, %v; want the stored key", key, ok)
	}

	t.Setenv(config.EnvFMPAPIKey, "env-key")
	key, ok = cfg.ResolveFMPAPIKey()
	if !ok || key != "env-key" {
		t.Errorf("ResolveFMPAPIKey() with env set = %q, %v; want the env key to override the stored one", key, ok)
	}
}

func TestFmpSetKeyRejectsEmpty(t *testing.T) {
	useTempDeviceConfigDir(t)

	if err := fmpSetKeyCmd.RunE(fmpSetKeyCmd, []string{""}); err == nil {
		t.Error("fmpSetKeyCmd.RunE(\"\") error = nil, want error")
	}
}
