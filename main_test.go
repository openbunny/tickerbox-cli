// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
)

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
