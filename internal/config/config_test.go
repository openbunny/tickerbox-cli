// SPDX-License-Identifier: MIT

package config

import (
	"io"
	"os"
	"strings"
	"testing"
)

func useTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir)
	return dir
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = w
	f()
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	os.Stderr = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}
	return string(out)
}

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	useTempConfigDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.Default != "" {
		t.Errorf("Default = %q, want empty", cfg.Default)
	}
	if len(cfg.Devices) != 0 {
		t.Errorf("Devices = %v, want empty", cfg.Devices)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := useTempConfigDir(t)

	cfg := &Config{Devices: map[string]Device{}}
	if err := cfg.Add("kitchen", "http://192.168.1.10"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := cfg.SetDefault("kitchen"); err != nil {
		t.Fatalf("SetDefault() error = %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	path, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("Path() = %q, want under %q", path, dir)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != configFilePerm {
		t.Errorf("file perm = %o, want %o", perm, configFilePerm)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Default != "kitchen" {
		t.Errorf("Default = %q, want kitchen", got.Default)
	}
	want := Device{Name: "kitchen", Host: "http://192.168.1.10"}
	if got.Devices["kitchen"] != want {
		t.Errorf("Devices[kitchen] = %+v, want %+v", got.Devices["kitchen"], want)
	}
}

func TestLoadWarnsOnGroupOrWorldReadablePermissions(t *testing.T) {
	tests := []struct {
		name     string
		perm     os.FileMode
		wantWarn bool
	}{
		{name: "owner only stays silent", perm: 0o600, wantWarn: false},
		{name: "group and world readable warns", perm: 0o644, wantWarn: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTempConfigDir(t)

			cfg := &Config{Devices: map[string]Device{}}
			if err := cfg.Add("kitchen", "http://10.0.0.1"); err != nil {
				t.Fatalf("Add() error = %v", err)
			}
			if err := cfg.Save(); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			path, err := Path()
			if err != nil {
				t.Fatalf("Path() error = %v", err)
			}
			if err := os.Chmod(path, tt.perm); err != nil {
				t.Fatalf("Chmod() error = %v", err)
			}

			stderr := captureStderr(t, func() {
				if _, err := Load(); err != nil {
					t.Fatalf("Load() error = %v", err)
				}
			})

			gotWarn := strings.Contains(stderr, "group- or world-readable")
			if gotWarn != tt.wantWarn {
				t.Errorf("stderr = %q, want warning = %v", stderr, tt.wantWarn)
			}
			if tt.wantWarn && !strings.Contains(stderr, path) {
				t.Errorf("stderr = %q, want it to name %q", stderr, path)
			}
		})
	}
}

func TestAddRemoveSetDefault(t *testing.T) {
	cfg := &Config{Devices: map[string]Device{}}

	if err := cfg.Add("", "host"); err == nil {
		t.Error("Add() with empty name: want error, got nil")
	}
	if err := cfg.Add("name", ""); err == nil {
		t.Error("Add() with empty host: want error, got nil")
	}

	if err := cfg.Add("living-room", "http://10.0.0.5"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if cfg.Default != "living-room" {
		t.Errorf("Default after first Add() = %q, want living-room (auto-selected)", cfg.Default)
	}
	if err := cfg.Add("office", "http://10.0.0.6"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if cfg.Default != "living-room" {
		t.Errorf("Default after second Add() = %q, want living-room (unchanged)", cfg.Default)
	}

	if err := cfg.SetDefault("unknown"); err == nil {
		t.Error("SetDefault(unknown): want error, got nil")
	}
	if err := cfg.SetDefault("office"); err != nil {
		t.Fatalf("SetDefault() error = %v", err)
	}
	if cfg.Default != "office" {
		t.Errorf("Default = %q, want office", cfg.Default)
	}

	got := cfg.List()
	want := []Device{{Name: "living-room", Host: "http://10.0.0.5"}, {Name: "office", Host: "http://10.0.0.6"}}
	if len(got) != len(want) {
		t.Fatalf("List() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	if err := cfg.Remove("unknown"); err == nil {
		t.Error("Remove(unknown): want error, got nil")
	}
	if err := cfg.Remove("office"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if cfg.Default != "" {
		t.Errorf("Default after removing default device = %q, want empty", cfg.Default)
	}
	if _, ok := cfg.Devices["office"]; ok {
		t.Error("Devices still contains removed device office")
	}
}

func TestAddRejectsSchemelessHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{name: "http scheme accepted", host: "http://tickerbox.local"},
		{name: "https scheme accepted", host: "https://tickerbox.local"},
		{name: "bare hostname rejected", host: "tickerbox.local", wantErr: true},
		{name: "bare IP rejected", host: "192.168.1.42", wantErr: true},
		{name: "other scheme rejected", host: "ws://tickerbox.local", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Devices: map[string]Device{}}
			err := cfg.Add("desk", tt.host)
			if tt.wantErr && err == nil {
				t.Fatalf("Add(%q) = nil error; want error", tt.host)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Add(%q) = %v; want nil", tt.host, err)
			}
		})
	}
}

func TestResolveHost(t *testing.T) {
	tests := []struct {
		name           string
		hostFlag       string
		deviceFlag     string
		envHost        string
		cfg            *Config
		want           string
		wantSource     HostSource
		wantDeviceName string
		wantErr        bool
	}{
		{
			name:           "host flag wins over everything",
			hostFlag:       "10.0.0.1",
			envHost:        "10.0.0.2",
			cfg:            &Config{Default: "kitchen", Devices: map[string]Device{"kitchen": {Name: "kitchen", Host: "10.0.0.3"}}},
			want:           "10.0.0.1",
			wantSource:     SourceHostFlag,
			wantDeviceName: "",
		},
		{
			name:           "device flag resolves to its host",
			deviceFlag:     "kitchen",
			envHost:        "10.0.0.2",
			cfg:            &Config{Devices: map[string]Device{"kitchen": {Name: "kitchen", Host: "10.0.0.3"}}},
			want:           "10.0.0.3",
			wantSource:     SourceDeviceFlag,
			wantDeviceName: "kitchen",
		},
		{
			name:       "unknown device flag errors",
			deviceFlag: "missing",
			cfg:        &Config{Devices: map[string]Device{}},
			wantErr:    true,
		},
		{
			name:           "env host wins over configured default",
			envHost:        "10.0.0.2",
			cfg:            &Config{Default: "kitchen", Devices: map[string]Device{"kitchen": {Name: "kitchen", Host: "10.0.0.3"}}},
			want:           "10.0.0.2",
			wantSource:     SourceEnv,
			wantDeviceName: "",
		},
		{
			name:           "configured default used when no flag or env",
			cfg:            &Config{Default: "kitchen", Devices: map[string]Device{"kitchen": {Name: "kitchen", Host: "10.0.0.3"}}},
			want:           "10.0.0.3",
			wantSource:     SourceDefault,
			wantDeviceName: "kitchen",
		},
		{
			name:    "no flags, no env, no default device: errors instead of a fallback host",
			cfg:     &Config{Devices: map[string]Device{}},
			wantErr: true,
		},
		{
			name:    "default device set but no longer configured: errors instead of a fallback host",
			cfg:     &Config{Default: "gone", Devices: map[string]Device{"kitchen": {Name: "kitchen", Host: "10.0.0.3"}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTempConfigDir(t)
			t.Setenv(envHost, tt.envHost)
			if err := tt.cfg.Save(); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			got, err := ResolveHost(tt.hostFlag, tt.deviceFlag)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ResolveHost() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveHost() error = %v", err)
			}
			if got.Host != tt.want {
				t.Errorf("ResolveHost().Host = %q, want %q", got.Host, tt.want)
			}
			if got.Source != tt.wantSource {
				t.Errorf("ResolveHost().Source = %v, want %v", got.Source, tt.wantSource)
			}
			if got.DeviceName != tt.wantDeviceName {
				t.Errorf("ResolveHost().DeviceName = %q, want %q", got.DeviceName, tt.wantDeviceName)
			}
		})
	}
}

func TestResolveHostErrorsAreDistinct(t *testing.T) {
	useTempConfigDir(t)

	t.Run("no selection", func(t *testing.T) {
		cfg := &Config{Devices: map[string]Device{}}
		_, err := cfg.Resolve("", "")
		if err == nil {
			t.Fatal("Resolve() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "no target device") {
			t.Errorf("Resolve() error = %q, want it to mention \"no target device\"", err.Error())
		}
	})

	t.Run("broken default", func(t *testing.T) {
		cfg := &Config{Default: "gone", Devices: map[string]Device{}}
		_, err := cfg.Resolve("", "")
		if err == nil {
			t.Fatal("Resolve() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "is not configured") {
			t.Errorf("Resolve() error = %q, want it to mention \"is not configured\"", err.Error())
		}
	})
}

func TestLoadAdoptsSoleDeviceAsDefault(t *testing.T) {
	tests := []struct {
		name        string
		devices     map[string]Device
		wantDefault string
	}{
		{name: "zero devices: no adoption", devices: map[string]Device{}, wantDefault: ""},
		{name: "one device: adopted as default", devices: map[string]Device{"kitchen": {Name: "kitchen", Host: "10.0.0.3"}}, wantDefault: "kitchen"},
		{
			name: "two devices: no adoption",
			devices: map[string]Device{
				"kitchen": {Name: "kitchen", Host: "10.0.0.3"},
				"office":  {Name: "office", Host: "10.0.0.4"},
			},
			wantDefault: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTempConfigDir(t)
			cfg := &Config{Devices: tt.devices}
			if err := cfg.Save(); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			got, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.Default != tt.wantDefault {
				t.Errorf("Load().Default = %q, want %q", got.Default, tt.wantDefault)
			}
		})
	}
}
