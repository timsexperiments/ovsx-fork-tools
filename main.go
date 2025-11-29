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
