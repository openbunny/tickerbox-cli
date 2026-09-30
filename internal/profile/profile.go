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

func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, appDirName, profilesDir)
}

func path(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid profile name %q", name)
	}
	return filepath.Join(Dir(), name+fileSuffix), nil
}

func List() ([]string, error) {
	entries, err := os.ReadDir(Dir())
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

func Load(name string) (*section.Snapshot, error) {
	p, err := path(name)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read profile %s: %w", name, err)
	}
	warnIfGroupOrWorldReadable(p)

	var s section.Snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("decode profile %s: %w", name, err)
	}
	return &s, nil
}

func Save(name string, s *section.Snapshot) error {
	p, err := path(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(Dir(), dirPerm); err != nil {
		return fmt.Errorf("create profiles directory: %w", err)
	}
	encoded, err := json.MarshalIndent(s, "", "  ")
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
