package main

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/golf/man"
	"code.linenisgreat.com/dodder/go/internal/victor/commands_dodder"
)

// Populated at link time by the fork's auto-injected -ldflags
// (-X main.version / -X main.commit). Must be at package scope.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	commands_dodder.SetVersion(version, commit)

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: dodder-gen_man <output-dir>\n")
		os.Exit(1)
	}

	outputDir := os.Args[1]

	config := man.PageConfig{
		BinaryName:  "dodder",
		Section:     1,
		Version:     version,
		Source:      "Dodder",
		Description: "distributed zettelkasten blob store",
		LongDescription: "Dodder is a distributed zettelkasten and " +
			"content-addressable blob store. It provides Git-like " +
			"version control for managing interconnected notes " +
			"(zettels) with sophisticated querying and remote " +
			"synchronization.",
	}

	if err := man.GenerateAll(
		config,
		commands_dodder.GetUtility("dodder"),
		outputDir,
	); err != nil {
		fmt.Fprintf(os.Stderr, "dodder-gen_man: %s\n", err)
		os.Exit(1)
	}
}
