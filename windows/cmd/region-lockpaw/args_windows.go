//go:build windows

package main

import "os"

func commandLineArguments() []string {
	if len(os.Args) < 2 {
		return nil
	}
	return os.Args[1:]
}
