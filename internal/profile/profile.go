// Package profile stores named device config snapshots as JSON files under
// the user's config directory, so a full section.Snapshot can be saved and
// re-applied later by name.
package profile

import (
	"encoding/json"
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
)

// Dir returns the directory profiles are stored in. If the OS cannot report
// a per-user config directory, it falls back to the OS temp directory rather
// than failing, since Dir itself cannot report an error.
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

// List returns the names of all saved profiles, sorted. A missing profiles
// directory is not an error: it returns an empty list, since no profile has
// been saved yet.
func List() ([]string, error) {
	entries, err := os.ReadDir(Dir())
	if os.IsNotExist(err) {
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

// Load reads the named profile.
func Load(name string) (*section.Snapshot, error) {
	p, err := path(name)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read profile %s: %w", name, err)
	}
	var s section.Snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("decode profile %s: %w", name, err)
	}
	return &s, nil
}

// Save writes s as the named profile, creating the profiles directory if
// needed. The file is written with 0600 permissions since a captured
// Snapshot may include wifi or ap credentials.
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

// Remove deletes the named profile.
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
