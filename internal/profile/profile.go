// SPDX-License-Identifier: MIT

package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openbunny/tickerbox-cli/internal/section"
)

const (
	appDirName  = "tickerbox"
	profilesDir = "profiles"
	fileSuffix  = ".json"
	dirPerm     = 0o700
	filePerm    = 0o600

	insecureReadBits = 0o044
)

type record struct {
	Description  string `json:"description,omitempty"`
	SourceDevice string `json:"source_device,omitempty"`
	section.Snapshot
}

func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine profiles directory: %w", err)
	}
	return filepath.Join(base, appDirName, profilesDir), nil
}

func path(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid profile name %q", name)
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+fileSuffix), nil
}

func List() ([]string, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != fileSuffix {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), fileSuffix))
	}
	sort.Strings(names)
	return names, nil
}

func loadRecord(name string) (record, error) {
	p, err := path(name)
	if err != nil {
		return record{}, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return record{}, fmt.Errorf("read profile %s: %w", name, err)
	}
	warnIfGroupOrWorldReadable(p)

	var r record
	if err := json.Unmarshal(raw, &r); err != nil {
		return record{}, fmt.Errorf("decode profile %s: %w", name, err)
	}
	return r, nil
}

func Load(name string) (*section.Snapshot, error) {
	r, err := loadRecord(name)
	if err != nil {
		return nil, err
	}
	return &r.Snapshot, nil
}

func LoadDescribed(name string) (*section.Snapshot, string, string, error) {
	r, err := loadRecord(name)
	if err != nil {
		return nil, "", "", err
	}
	return &r.Snapshot, r.Description, r.SourceDevice, nil
}

func LoadWithSource(name string) (*section.Snapshot, string, error) {
	r, err := loadRecord(name)
	if err != nil {
		return nil, "", err
	}
	return &r.Snapshot, r.SourceDevice, nil
}

func LoadDescription(name string) (string, error) {
	r, err := loadRecord(name)
	if err != nil {
		return "", err
	}
	return r.Description, nil
}

func SaveDescribed(name string, s *section.Snapshot, description, sourceDevice string) error {
	p, err := path(name)
	if err != nil {
		return err
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create profiles directory: %w", err)
	}
	encoded, err := json.MarshalIndent(record{Description: description, SourceDevice: sourceDevice, Snapshot: *s}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile %s: %w", name, err)
	}
	if err := os.WriteFile(p, encoded, filePerm); err != nil {
		return fmt.Errorf("write profile %s: %w", name, err)
	}
	return nil
}

func warnIfGroupOrWorldReadable(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if perm := info.Mode().Perm(); perm&insecureReadBits != 0 {
		fmt.Fprintf(os.Stderr, "warning: profile file %s is group- or world-readable (mode %o); run chmod %o %s to restrict it\n", path, perm, filePerm, path)
	}
}

// Exists reports whether a profile named name is already saved.
func Exists(name string) (bool, error) {
	p, err := path(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("stat profile %s: %w", name, err)
	default:
		return true, nil
	}
}

func Remove(name string) error {
	p, err := path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		return fmt.Errorf("remove profile %s: %w", name, err)
	}
	return nil
}
