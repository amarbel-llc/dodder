package main

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/uniform/commands_dodder"
)

// Populated at link time by the fork's auto-injected -ldflags
// (-X main.version / -X main.commit). Must be at package scope.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	commands_dodder.SetVersion(version, commit)
	utility := commands_dodder.GetUtility("dodder")
	utility.Run(os.Args)
}
