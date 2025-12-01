// Package workflows contains the GitHub Actions workflow templates used by the setup tool.
// These workflows are embedded into the binary and written to the user's repository during setup.
package workflows

import (
	_ "embed"
)

//go:embed check-version.yml
var CheckVersion []byte

//go:embed release.yml
var Release []byte

//go:embed sync.yml
var Sync []byte
