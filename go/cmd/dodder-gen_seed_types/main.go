// Command dodder-gen_seed_types generates the dodder.net seed-set type files
// (FDR-0010 Phase 3): one hyphence .type file per generic file-format type
// from the table in table.go, emitted into zz-seed/types/ at the repo root.
// Regeneration is deterministic and idempotent (stable ordering, no
// timestamps); stale .type files from removed table entries are pruned. Not
// wired into dodder's CLI — run via `just generate-seed-types` from the repo
// root.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := flag.String(
		"dir",
		"zz-seed/types",
		"output directory for the generated .type files",
	)

	flag.Parse()

	if err := generateSeedTypes(*dir); err != nil {
		fmt.Fprintf(os.Stderr, "dodder-gen_seed_types: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(
		os.Stderr,
		"wrote %d type files to %s\n",
		len(seedTypes),
		*dir,
	)
}

// generateSeedTypes writes one <name>.type file per table entry into dir and
// prunes any stale .type files left over from removed entries, so repeated
// runs converge on exactly the table's file set.
func generateSeedTypes(dir string) (err error) {
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	expected := make(map[string]bool, len(seedTypes))

	for _, entry := range seedTypes {
		fileName := entry.Name + ".type"
		expected[fileName] = true

		if err = os.WriteFile(
			filepath.Join(dir, fileName),
			entry.render(),
			0o644,
		); err != nil {
			return err
		}
	}

	var dirEntries []os.DirEntry

	if dirEntries, err = os.ReadDir(dir); err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		name := dirEntry.Name()

		if !strings.HasSuffix(name, ".type") || expected[name] {
			continue
		}

		if err = os.Remove(filepath.Join(dir, name)); err != nil {
			return err
		}
	}

	return err
}
