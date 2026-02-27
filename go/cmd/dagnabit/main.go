package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/lib/_/dagnabit"
)

func main() {
	var dryRun bool
	var verbose bool

	flag.BoolVar(&dryRun, "n", false, "show what would be moved without moving")
	flag.BoolVar(&dryRun, "dry-run", false, "show what would be moved without moving")
	flag.BoolVar(&verbose, "v", false, "enable verbose output")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose output")
	flag.Parse()

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	goModPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: must be run from a directory containing go.mod\n")
		os.Exit(1)
	}

	r := dagnabit.Repositioner{
		Reader: dagnabit.GoListReader{
			ModulePath:      "code.linenisgreat.com/dodder/go",
			Dir:             dir,
			PackagePrefixes: []string{"lib", "internal"},
		},
		Mapper:  dagnabit.MakeNATOLevelMapper(),
		Mover:   dagnabit.JustMover{Dir: dir},
		DryRun:  dryRun,
		Verbose: verbose,
	}

	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
