// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
)

func TestVersionFlagUsesBuildMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, flags, want string
	}{
		{name: "release", flags: "-X github.com/openbunny/tickerbox-cli/cmd.version=1.2.3", want: "1.2.3"},
		{name: "development"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "tickerbox")
			if output, err := exec.Command("go", "build", "-ldflags", tc.flags, "-o", binary, ".").CombinedOutput(); err != nil {
				t.Fatalf("build binary: %v\n%s", err, output)
			}
			want := tc.want
			if want == "" {
				output, err := exec.Command(binary, "version").CombinedOutput()
				if err != nil {
					t.Fatalf("read development version: %v\n%s", err, output)
				}
				fields := strings.Fields(string(output))
				if len(fields) < 2 {
					t.Fatalf("version output %q has no version", output)
				}
				want = fields[1]
			}
			for _, argument := range []string{"--version", "version"} {
				output, err := exec.Command(binary, argument).CombinedOutput()
				if err != nil {
					t.Fatalf("run %s: %v\n%s", argument, err, output)
				}
				if !strings.Contains(string(output), want) || strings.Contains(string(output), "unknown (built from source)") {
					t.Errorf("%s = %q; want version %q", argument, output, want)
				}
			}
		})
	}
}

func TestFangExecuteStylesAndPropagatesError(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	wantErr := errors.New("boom")
	throwaway := &cobra.Command{
		Use: "throwaway",
		RunE: func(cmd *cobra.Command, args []string) error {
			return wantErr
		},
	}
	var out bytes.Buffer
	throwaway.SetOut(&out)
	throwaway.SetErr(&out)

	err := fang.Execute(context.Background(), throwaway)
	if !errors.Is(err, wantErr) {
		t.Fatalf("fang.Execute() error = %v, want %v", err, wantErr)
	}
	if !bytes.ContainsRune(out.Bytes(), '\x1b') {
		t.Errorf("fang.Execute() error output = %q, want styled (ANSI-escaped) output", out.String())
	}
}

func TestFangExecuteSuccess(t *testing.T) {
	throwaway := &cobra.Command{
		Use: "throwaway",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	var out bytes.Buffer
	throwaway.SetOut(&out)
	throwaway.SetErr(&out)

	if err := fang.Execute(context.Background(), throwaway); err != nil {
		t.Fatalf("fang.Execute() error = %v, want nil", err)
	}
}
