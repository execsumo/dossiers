package main

import (
	"dossier/internal/cli"
)

// version is set at release build time via -ldflags "-X main.version=...".
// It defaults to "dev" for local/unstamped builds.
var version = "dev"

func main() {
	cli.Version = version
	cli.Execute()
}
