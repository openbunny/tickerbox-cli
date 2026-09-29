// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

const (
	helpGoldenDir  = "testdata/help"
	helpFilePerm   = 0o644
	helpRootGolden = "root"
)

var updateGolden = flag.Bool("update", false, "rewrite golden fixtures in "+helpGoldenDir)

type helpCase struct {
	args   []string
	golden string
}

func collectHelpCases(cmd *cobra.Command, args []string, slug string) []helpCase {
	if slug == "" {
		slug = helpRootGolden
	}

	caseArgs := append(append([]string{}, args...), "--help")
	cases := []helpCase{{args: caseArgs, golden: slug + ".golden"}}

	for _, sub := range cmd.Commands() {
		childArgs := append(append([]string{}, args...), sub.Name())
		childSlug := sub.Name()
		if slug != helpRootGolden {
			childSlug = slug + "_" + sub.Name()
		}
		cases = append(cases, collectHelpCases(sub, childArgs, childSlug)...)
	}

	return cases
}

func TestHelpGolden(t *testing.T) {
	for _, tc := range collectHelpCases(rootCmd, nil, "") {
		t.Run(tc.golden, func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)
			rootCmd.SetArgs(tc.args)

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("execute %v: %v", tc.args, err)
			}

			got := out.Bytes()
			path := filepath.Join(helpGoldenDir, tc.golden)

			if *updateGolden {
				if err := os.WriteFile(path, got, helpFilePerm); err != nil {
					t.Fatalf("write golden %s: %v", path, err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden %s: %v (run go test ./cmd/ -run TestHelpGolden -update to generate it)", path, err)
			}

			if !bytes.Equal(normalizeNewlines(got), normalizeNewlines(want)) {
				t.Errorf("help output for %v does not match %s\n--- got ---\n%s\n--- want ---\n%s", tc.args, path, got, want)
			}
		})
	}
}

func normalizeNewlines(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
