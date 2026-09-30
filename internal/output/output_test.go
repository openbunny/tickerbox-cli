// SPDX-License-Identifier: MIT

package output

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return string(out)
}

func TestEmitJSON(t *testing.T) {
	got := captureStdout(t, func() {
		if err := EmitJSON(map[string]int{"a": 1}); err != nil {
			t.Fatalf("EmitJSON: %v", err)
		}
	})
	want := "{\n  \"a\": 1\n}\n"
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestEmitJSONError(t *testing.T) {
	err := EmitJSON(make(chan int))
	if err == nil {
		t.Fatal("expected error for unencodable value, got nil")
	}
}

func TestKV(t *testing.T) {
	got := captureStdout(t, func() {
		if err := KV([][2]string{{"name", "tickerbox"}, {"host", "tickerbox.local"}}); err != nil {
			t.Fatalf("KV: %v", err)
		}
	})
	for _, want := range []string{"name:", "tickerbox", "host:", "tickerbox.local"} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q missing %q", got, want)
		}
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("got %d lines; want 2: %q", len(lines), got)
	}
}

func TestKVEmpty(t *testing.T) {
	got := captureStdout(t, func() {
		if err := KV(nil); err != nil {
			t.Fatalf("KV: %v", err)
		}
	})
	if got != "" {
		t.Errorf("got %q; want empty output", got)
	}
}

func TestTable(t *testing.T) {
	got := captureStdout(t, func() {
		if err := Table([]string{"NAME", "HOST"}, [][]string{{"a", "1.2.3.4"}, {"b", "tickerbox.local"}}); err != nil {
			t.Fatalf("Table: %v", err)
		}
	})
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines; want 3 (header + 2 rows): %q", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "NAME") || !strings.Contains(lines[0], "HOST") {
		t.Errorf("header line %q missing expected columns", lines[0])
	}
	if !strings.Contains(lines[1], "a") || !strings.Contains(lines[1], "1.2.3.4") {
		t.Errorf("row 1 %q missing expected columns", lines[1])
	}
}

func TestTableNoRows(t *testing.T) {
	got := captureStdout(t, func() {
		if err := Table([]string{"NAME"}, nil); err != nil {
			t.Fatalf("Table: %v", err)
		}
	})
	if strings.TrimRight(got, "\n") != "NAME" {
		t.Errorf("got %q; want header-only output", got)
	}
}
