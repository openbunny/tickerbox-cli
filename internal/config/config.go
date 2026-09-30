// SPDX-License-Identifier: MIT

package config

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	appDirName     = "tickerbox"
	configFileName = "config.toml"
	configDirPerm  = 0o700
	configFilePerm = 0o600

	envHost = "TICKERBOX_HOST"

	insecureReadBits = 0o044
)

type Device struct {
	Name string
	Host string
}

type Config struct {
	Default string
	Devices map[string]Device
}

func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine config directory: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

func Load() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, configFileName)

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return &Config{Devices: map[string]Device{}}, nil
	case err != nil:
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	warnIfGroupOrWorldReadable(path)

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Devices == nil {
		cfg.Devices = map[string]Device{}
	}
	if cfg.Default == "" && len(cfg.Devices) == 1 {
		for name := range cfg.Devices {
			cfg.Default = name
		}
	}
	return &cfg, nil
}

func warnIfGroupOrWorldReadable(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if perm := info.Mode().Perm(); perm&insecureReadBits != 0 {
		fmt.Fprintf(os.Stderr, "warning: config file %s is group- or world-readable (mode %o); run chmod %o %s to restrict it\n", path, perm, configFilePerm, path)
	}
}

func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, configDirPerm); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, data, configFilePerm); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

func (c *Config) List() []Device {
	devices := slices.Collect(maps.Values(c.Devices))
	slices.SortFunc(devices, func(a, b Device) int { return cmp.Compare(a.Name, b.Name) })
	return devices
}

func (c *Config) Add(name, host string) error {
	if name == "" {
		return errors.New("device name must not be empty")
	}
	if host == "" {
		return errors.New("device host must not be empty")
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		return fmt.Errorf("device host %q must include a scheme (http:// or https://)", host)
	}
	if c.Devices == nil {
		c.Devices = map[string]Device{}
	}
	c.Devices[name] = Device{Name: name, Host: host}
	if c.Default == "" {
		c.Default = name
	}
	return nil
}

func (c *Config) Remove(name string) error {
	if _, ok := c.Devices[name]; !ok {
		return fmt.Errorf("unknown device %q", name)
	}
	delete(c.Devices, name)
	if c.Default == name {
		c.Default = ""
	}
	return nil
}

func (c *Config) SetDefault(name string) error {
	if _, ok := c.Devices[name]; !ok {
		return fmt.Errorf("unknown device %q", name)
	}
	c.Default = name
	return nil
}

type HostSource int

const (
	SourceHostFlag HostSource = iota
	SourceDeviceFlag
	SourceEnv
	SourceDefault
)

type ResolvedHost struct {
	Host       string
	Source     HostSource
	DeviceName string
}

// Resolve implements the target-host precedence chain: --host, then --device, then
// $TICKERBOX_HOST, then the configured default device. Unlike the deleted
// package-level ResolveHost, it never falls back to a hardcoded default host: with
// nothing to resolve, or a default device the config no longer lists, it returns an
// error naming the fix instead of dying later on a fallback host that fooled nobody.
func (c *Config) Resolve(hostFlag, deviceFlag string) (ResolvedHost, error) {
	if hostFlag != "" {
		return ResolvedHost{Host: hostFlag, Source: SourceHostFlag}, nil
	}

	if deviceFlag != "" {
		d, ok := c.Devices[deviceFlag]
		if !ok {
			return ResolvedHost{}, fmt.Errorf("unknown device %q: run `tickerbox device list` to see configured devices", deviceFlag)
		}
		return ResolvedHost{Host: d.Host, Source: SourceDeviceFlag, DeviceName: deviceFlag}, nil
	}

	if host := os.Getenv(envHost); host != "" {
		return ResolvedHost{Host: host, Source: SourceEnv}, nil
	}

	if c.Default != "" {
		d, ok := c.Devices[c.Default]
		if !ok {
			path, pathErr := Path()
			if pathErr != nil {
				return ResolvedHost{}, pathErr
			}
			return ResolvedHost{}, fmt.Errorf("default device %q is not configured\n\n"+
				"config at %s lists it as default but has no matching device entry; run `tickerbox device use <name>` to pick one, or edit the file to clear the stale default",
				c.Default, path)
		}
		return ResolvedHost{Host: d.Host, Source: SourceDefault, DeviceName: c.Default}, nil
	}

	return ResolvedHost{}, errors.New("no target device: no --host, no --device, no $TICKERBOX_HOST, and no default device configured\n\n" +
		"add one: `tickerbox device add <name> <host>`\n" +
		"set a default: `tickerbox device use <name>`\n" +
		"or target one call: pass --host <url>")
}

func ResolveHost(hostFlag, deviceFlag string) (ResolvedHost, error) {
	cfg, err := Load()
	if err != nil {
		return ResolvedHost{}, err
	}
	return cfg.Resolve(hostFlag, deviceFlag)
}
