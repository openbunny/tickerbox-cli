// Package config manages the multi-device TOML config file that maps a
// device name to a host, so commands can target a device by name instead of
// a raw address.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"

	"github.com/openbunny/tickerbox-cli/internal/client"
)

const (
	appDirName     = "tickerbox"
	configFileName = "config.toml"
	configDirPerm  = 0o700
	configFilePerm = 0o600

	// envHost overrides the host when no --host or --device flag is given.
	envHost = "TICKERBOX_HOST"
)

// Device is one named target the CLI can send requests to.
type Device struct {
	Name string
	Host string
}

// Config is the on-disk, multi-device configuration: a set of named devices
// and which one is used when no device is specified explicitly.
type Config struct {
	Default string
	Devices map[string]Device
}

// Path returns the config file's location. It cannot report an error
// because os.UserConfigDir failing means the environment has no usable home
// directory; in that case Path falls back to a path relative to the current
// directory, and Load or Save will fail with the underlying cause when they
// actually try to use it.
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

// Load reads the config file. A missing file is not an error: it returns an
// empty Config, since a device set up for the first time has none yet.
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

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Devices == nil {
		cfg.Devices = map[string]Device{}
	}
	return &cfg, nil
}

// Save writes the config file, creating its parent directory if needed. The
// file is written with 0600 permissions since it may hold device hosts on a
// private network.
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

// List returns the configured devices sorted by name.
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

// Add sets or replaces the device named name with the given host.
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

// Remove deletes the device named name. It errors if no such device exists.
// If name was the default device, the default is cleared.
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

// SetDefault marks name as the default device. It errors if no such device
// exists.
func (c *Config) SetDefault(name string) error {
	if _, ok := c.Devices[name]; !ok {
		return fmt.Errorf("unknown device %q", name)
	}
	c.Default = name
	return nil
}

// ResolveHost picks the device host a command should use, in order:
// hostFlag if set, then devices[deviceFlag].Host if deviceFlag is set (an
// unknown device is an error), then the TICKERBOX_HOST environment
// variable, then the configured default device, then client.DefaultHost.
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
