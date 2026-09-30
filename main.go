// SPDX-License-Identifier: MIT

//go:generate go run ./docs/gen

package main

import (
	"context"
	"os"

	"charm.land/fang/v2"

	"github.com/openbunny/tickerbox-cli/cmd"
)

func main() {
	if err := fang.Execute(context.Background(), cmd.Root()); err != nil {
		os.Exit(1)
	}
}
