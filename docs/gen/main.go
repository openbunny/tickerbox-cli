// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/spf13/cobra/doc"

	"github.com/openbunny/tickerbox-cli/cmd"
)

const (
	manSection = 1
	dirPerm    = 0o755
)

func main() {
	man := flag.Bool("man", false, "generate man pages instead of markdown")
	flag.Parse()

	_, self, _, _ := runtime.Caller(0)
	genDir := filepath.Dir(self)
	docsDir := filepath.Dir(genDir)
	repoRoot := filepath.Dir(docsDir)

	root := cmd.Root()
	root.DisableAutoGenTag = true
	root.InitDefaultCompletionCmd()

	if *man {
		manDir := filepath.Join(repoRoot, "dist", "man")
		if err := os.MkdirAll(manDir, dirPerm); err != nil {
			log.Fatal(err)
		}
		header := &doc.GenManHeader{Title: "TICKERBOX", Section: strconv.Itoa(manSection)}
		if err := doc.GenManTree(root, header, manDir); err != nil {
			log.Fatal(err)
		}
		return
	}

	markdownDir := filepath.Join(docsDir, "cli")
	if err := os.MkdirAll(markdownDir, dirPerm); err != nil {
		log.Fatal(err)
	}
	if err := doc.GenMarkdownTree(root, markdownDir); err != nil {
		log.Fatal(err)
	}
}
