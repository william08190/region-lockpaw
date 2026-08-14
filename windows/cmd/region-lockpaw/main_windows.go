//go:build windows

package main

import (
	"os"
	"slices"

	"github.com/william08190/region-lockpaw/windows/internal/platform"
)

var version = "dev"

func main() {
	arguments := commandLineArguments()
	if err := platform.Run(version, arguments); err != nil {
		if slices.Contains(arguments, "--self-test") {
			os.Exit(1)
		}
		platform.ShowFatalError(err)
	}
}
