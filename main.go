// Package main provides the entry point for the ovsx-setup tool.
//
// Usage:
//
//	ovsx-setup -p <publisher> -e <extension_path>
//
// Options:
//
//	-p <publisher>	The publisher name for the extension.
//	-e <extension_path>	The path to the extension relative to the cwd.
package main

import (
	"fmt"
	"os"

	app "github.com/timsexperiments/ovsx-fork-tools/internal/setup"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
