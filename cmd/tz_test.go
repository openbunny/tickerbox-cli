// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/openbunny/tickerbox-cli/internal/tz"
)

func runTzList(t *testing.T, grep string, jsonMode bool) string {
	t.Helper()
	origGrep, origJSON := tzListGrep, jsonFlag
	t.Cleanup(func() { tzListGrep, jsonFlag = origGrep, origJSON })
	tzListGrep, jsonFlag = grep, jsonMode

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	runErr := tzListCmd.RunE(tzListCmd, nil)

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}

	if runErr != nil {
		t.Fatalf("tzListCmd.RunE() = %v", runErr)
	}
	return buf.String()
}

func TestTzListNoFilterJSONMatchesLoad(t *testing.T) {
	out := runTzList(t, "", true)

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output %q: %v", out, err)
	}
	want := tz.Load()
	if len(got) != len(want) {
		t.Fatalf("got %d entries; want %d", len(got), len(want))
	}
	for label, posix := range want {
		if got[label] != posix {
			t.Errorf("label %q: got %q; want %q", label, got[label], posix)
		}
	}
}

func TestTzListGrepFiltersCaseInsensitively(t *testing.T) {
	all := tz.Load()
	var wantLabels []string
	for label := range all {
		if strings.Contains(strings.ToLower(label), "europe") {
			wantLabels = append(wantLabels, label)
		}
	}
	if len(wantLabels) == 0 {
		t.Skip("embedded timezones.json has no Europe/* labels to filter against")
	}

	out := runTzList(t, "EUROPE", true)

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output %q: %v", out, err)
	}
	if len(got) != len(wantLabels) {
		t.Errorf("got %d filtered entries; want %d", len(got), len(wantLabels))
	}
	for label := range got {
		if !strings.Contains(strings.ToLower(label), "europe") {
			t.Errorf("filtered result contains non-matching label %q", label)
		}
	}
}

func TestTzListGrepNoMatch(t *testing.T) {
	out := runTzList(t, "no-such-region-xyz", true)
	if strings.TrimSpace(out) != "{}" {
		t.Errorf("got %q; want an empty JSON object for a filter matching nothing", out)
	}
}

func TestTzListTableOutputHasHeader(t *testing.T) {
	out := runTzList(t, "", false)
	if !strings.HasPrefix(out, "LABEL") {
		t.Errorf("table output %q does not start with the LABEL header", out)
	}
}
