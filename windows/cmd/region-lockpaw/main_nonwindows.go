//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("Region Lockpaw for Windows must be run on Windows 10 or later.")
}
