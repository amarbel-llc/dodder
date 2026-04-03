package main

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/golf/man"
	"code.linenisgreat.com/dodder/go/internal/india/commands_madder"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: madder-gen_man <output-dir>\n")
		os.Exit(1)
	}

	outputDir := os.Args[1]

	config := man.PageConfig{
		BinaryName:  "madder",
		Section:     1,
		Version:     "0.0.1",
		Source:      "Dodder",
		Description: "content-addressable blob store operations",
		LongDescription: "Madder provides low-level operations for " +
			"content-addressable blob stores, including reading, " +
			"writing, packing, and synchronizing blobs.",
	}

	if err := man.GenerateAll(
		config,
		commands_madder.GetUtility(),
		outputDir,
	); err != nil {
		fmt.Fprintf(os.Stderr, "madder-gen_man: %s\n", err)
		os.Exit(1)
	}
}
