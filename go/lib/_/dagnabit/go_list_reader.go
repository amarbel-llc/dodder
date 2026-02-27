package dagnabit

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TODO: refactor to use golang.org/x/tools/go/packages for direct
// programmatic access to the import graph instead of shelling out.

// GoListReader reads Go package dependencies by shelling out to `go list`.
// ModulePath is the Go module path (e.g., "code.linenisgreat.com/dodder/go").
// Dir is the working directory to run `go list` from.
// PackagePrefixes are directory prefixes containing packages (e.g., ["lib", "internal"]).
// Node names in returned edges include the prefix (e.g., "lib/_/ohio_buffer").
type GoListReader struct {
	ModulePath      string
	Dir             string
	PackagePrefixes []string
}

func (r GoListReader) ReadDependencies() ([]Edge, error) {
	var allEdges []Edge

	for _, prefix := range r.PackagePrefixes {
		edges, err := r.readPrefix(prefix)
		if err != nil {
			return nil, err
		}

		allEdges = append(allEdges, edges...)
	}

	// Deduplicate edges — go list produces duplicates when multiple files
	// in the same package import the same dependency.
	seen := make(map[Edge]struct{})
	deduped := make([]Edge, 0, len(allEdges))

	for _, e := range allEdges {
		if _, ok := seen[e]; ok {
			continue
		}

		seen[e] = struct{}{}
		deduped = append(deduped, e)
	}

	return deduped, nil
}

func (r GoListReader) readPrefix(prefix string) ([]Edge, error) {
	patterns, err := listPatternsForPrefix(r.Dir, prefix)
	if err != nil {
		return nil, err
	}

	args := append(
		[]string{"list", "-f", `{{.ImportPath}}{{"\t"}}{{range .Imports}}{{.}} {{end}}`},
		patterns...,
	)

	cmd := exec.Command("go", args...)
	cmd.Dir = r.Dir

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %w", prefix, err)
	}

	modulePrefix := r.ModulePath + "/"
	var edges []Edge

	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		sourceFull := parts[0]

		if !strings.HasPrefix(sourceFull, modulePrefix) {
			continue
		}

		source := trimToThreeComponents(
			strings.TrimPrefix(sourceFull, modulePrefix),
		)

		if source == "" {
			continue
		}

		imports := strings.Fields(parts[1])

		for _, imp := range imports {
			if !strings.HasPrefix(imp, modulePrefix) {
				continue
			}

			target := trimToThreeComponents(
				strings.TrimPrefix(imp, modulePrefix),
			)

			if target == "" || target == source {
				continue
			}

			edges = append(edges, Edge{Source: source, Target: target})
		}
	}

	return edges, scanner.Err()
}

// listPatternsForPrefix returns go list patterns that cover all packages
// under prefix, including _-prefixed directories that go list's ...
// wildcard skips by convention.
func listPatternsForPrefix(dir, prefix string) ([]string, error) {
	patterns := []string{fmt.Sprintf("./%s/...", prefix)}

	prefixDir := filepath.Join(dir, prefix)

	topEntries, err := os.ReadDir(prefixDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", prefixDir, err)
	}

	for _, topEntry := range topEntries {
		name := topEntry.Name()
		if !topEntry.IsDir() || !strings.HasPrefix(name, "_") {
			continue
		}

		underscoreDir := filepath.Join(prefixDir, name)

		subEntries, err := os.ReadDir(underscoreDir)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", underscoreDir, err)
		}

		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}

			patterns = append(
				patterns,
				fmt.Sprintf("./%s/%s/%s", prefix, name, subEntry.Name()),
			)
		}
	}

	return patterns, nil
}

// trimToThreeComponents returns the first three path components (e.g.,
// "lib/alfa/errors/context" -> "lib/alfa/errors"). Returns "" if fewer than 3.
func trimToThreeComponents(path string) string {
	parts := strings.SplitN(path, "/", 4)
	if len(parts) < 3 {
		return ""
	}

	return parts[0] + "/" + parts[1] + "/" + parts[2]
}

// trimToTwoComponents returns the first two path components (e.g.,
// "alfa/errors/context" -> "alfa/errors"). Returns "" if fewer than 2.
func trimToTwoComponents(path string) string {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return ""
	}

	return parts[0] + "/" + parts[1]
}
