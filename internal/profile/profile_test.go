// SPDX-License-Identifier: MIT

package profile

import (
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/section"
	"github.com/openbunny/tickerbox-cli/internal/tickers"
)

func withTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("AppData", home)
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

func testSnapshot() *section.Snapshot {
	return &section.Snapshot{
		Display: map[string]any{"brightness": float64(200)},
		Tickers: []tickers.Entry{
			{Type: tickers.TypeStocks, Ticker: "AAPL", Time: tickers.Time15Min, Currency: tickers.CurrencyUSD},
		},
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	withTempHome(t)

	want := testSnapshot()
	if err := Save("office", want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load("office")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v; want %+v", got, want)
	}
}

func TestSaveWritesOwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not carry Unix permission bits")
	}
	withTempHome(t)

	if err := Save("secure", testSnapshot()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	p, err := path("secure")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != filePerm {
		t.Errorf("got file mode %o; want %o", got, filePerm)
	}
}

func TestLoadWarnsOnGroupOrWorldReadablePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not carry Unix permission bits")
	}
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
			withTempHome(t)

			if err := Save("office", testSnapshot()); err != nil {
				t.Fatalf("Save: %v", err)
			}
			p, err := path("office")
			if err != nil {
				t.Fatalf("path: %v", err)
			}
			if err := os.Chmod(p, tt.perm); err != nil {
				t.Fatalf("Chmod() error = %v", err)
			}

			stderr := captureStderr(t, func() {
				if _, err := Load("office"); err != nil {
					t.Fatalf("Load() error = %v", err)
				}
			})

			gotWarn := strings.Contains(stderr, "group- or world-readable")
			if gotWarn != tt.wantWarn {
				t.Errorf("stderr = %q, want warning = %v", stderr, tt.wantWarn)
			}
			if tt.wantWarn && !strings.Contains(stderr, p) {
				t.Errorf("stderr = %q, want it to name %q", stderr, p)
			}
		})
	}
}

func TestList(t *testing.T) {
	withTempHome(t)

	if names, err := List(); err != nil || len(names) != 0 {
		t.Fatalf("List() on empty dir = %v, %v; want [], nil", names, err)
	}

	for _, name := range []string{"beta", "alpha"} {
		if err := Save(name, testSnapshot()); err != nil {
			t.Fatalf("Save(%s): %v", name, err)
		}
	}

	names, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("List() = %v; want %v (sorted)", names, want)
	}
}

func TestRemove(t *testing.T) {
	withTempHome(t)

	if err := Save("temp", testSnapshot()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Remove("temp"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := Load("temp"); err == nil {
		t.Fatal("Load after Remove: expected error, got nil")
	}
}

func TestRemoveMissingProfile(t *testing.T) {
	withTempHome(t)

	if err := Remove("nonexistent"); err == nil {
		t.Fatal("expected error removing a profile that does not exist, got nil")
	}
}

func TestLoadMissingProfile(t *testing.T) {
	withTempHome(t)

	if _, err := Load("nonexistent"); err == nil {
		t.Fatal("expected error loading a profile that does not exist, got nil")
	}
}

func TestRejectsUnsafeNames(t *testing.T) {
	withTempHome(t)

	for _, name := range []string{"", "..", "../escape", "a/b"} {
		if _, err := Load(name); err == nil {
			t.Errorf("Load(%q): expected error, got nil", name)
		}
		if err := Save(name, testSnapshot()); err == nil {
			t.Errorf("Save(%q): expected error, got nil", name)
		}
		if err := Remove(name); err == nil {
			t.Errorf("Remove(%q): expected error, got nil", name)
		}
	}
}
