// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/pelletier/go-toml/v2"

	"github.com/openbunny/tickerbox-cli/internal/client"
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

func Path() string {
	dir, err := configDir()
	if err != nil {
		return filepath.Join(appDirName, configFileName)
	}
	return filepath.Join(dir, configFileName)
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
	return &cfg, nil
}

func warnIfGroupOrWorldReadable(path string) {
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if perm := info.Mode().Perm(); perm&insecureReadBits != 0 {
		fmt.Fprintf(os.Stderr, "warning: config file %s is group- or world-readable (mode %o)\n", path, perm)
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
	devices := make([]Device, 0, len(c.Devices))
	for _, d := range c.Devices {
		devices = append(devices, d)
	}
	slices.SortFunc(devices, func(a, b Device) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return devices
}

func (c *Config) Add(name, host string) error {
	if name == "" {
		return errors.New("device name must not be empty")
	}
	if host == "" {
		return errors.New("device host must not be empty")
	}
	if c.Devices == nil {
		c.Devices = map[string]Device{}
	}
	c.Devices[name] = Device{Name: name, Host: host}
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

func ResolveHost(hostFlag, deviceFlag string) (string, error) {
	if hostFlag != "" {
		return hostFlag, nil
	}

	cfg, err := Load()
	if err != nil {
		return "", err
	}

	if deviceFlag != "" {
		d, ok := cfg.Devices[deviceFlag]
		if !ok {
			return "", fmt.Errorf("unknown device %q", deviceFlag)
		}
		return d.Host, nil
	}

	if host := os.Getenv(envHost); host != "" {
		return host, nil
	}

	if cfg.Default != "" {
		if d, ok := cfg.Devices[cfg.Default]; ok {
			return d.Host, nil
		}
	}

	return client.DefaultHost, nil
}
