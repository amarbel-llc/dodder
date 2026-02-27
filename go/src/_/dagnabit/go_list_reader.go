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
// PackagePrefix is the directory prefix for internal packages (e.g., "src").
type GoListReader struct {
	ModulePath    string
	Dir           string
	PackagePrefix string
}

func (r GoListReader) ReadDependencies() ([]Edge, error) {
	patterns, err := r.listPatterns()
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
		return nil, fmt.Errorf("go list: %w", err)
	}

	internalPrefix := r.ModulePath + "/" + r.PackagePrefix + "/"
	var edges []Edge

	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		sourceFull := parts[0]

		if !strings.HasPrefix(sourceFull, internalPrefix) {
			continue
		}

		source := trimToTwoComponents(
			strings.TrimPrefix(sourceFull, internalPrefix),
		)

		if source == "" {
			continue
		}

		imports := strings.Fields(parts[1])

		for _, imp := range imports {
			if !strings.HasPrefix(imp, internalPrefix) {
				continue
			}

			target := trimToTwoComponents(
				strings.TrimPrefix(imp, internalPrefix),
			)

			if target == "" || target == source {
				continue
			}

			edges = append(edges, Edge{Source: source, Target: target})
		}
	}

	// Deduplicate edges — go list produces duplicates when multiple files
	// in the same package import the same dependency.
	seen := make(map[Edge]struct{})
	deduped := make([]Edge, 0, len(edges))

	for _, e := range edges {
		if _, ok := seen[e]; ok {
			continue
		}

		seen[e] = struct{}{}
		deduped = append(deduped, e)
	}

	return deduped, scanner.Err()
}

// listPatterns returns go list patterns that cover all packages under
// PackagePrefix, including _-prefixed directories that go list's ...
// wildcard skips by convention. Since go list ignores _-prefixed dirs
// even with explicit ./_/... patterns, each sub-package must be listed
// individually.
func (r GoListReader) listPatterns() ([]string, error) {
	patterns := []string{fmt.Sprintf("./%s/...", r.PackagePrefix)}

	srcDir := filepath.Join(r.Dir, r.PackagePrefix)

	topEntries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", srcDir, err)
	}

	for _, topEntry := range topEntries {
		name := topEntry.Name()
		if !topEntry.IsDir() || !strings.HasPrefix(name, "_") {
			continue
		}

		underscoreDir := filepath.Join(srcDir, name)

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
				fmt.Sprintf("./%s/%s/%s", r.PackagePrefix, name, subEntry.Name()),
			)
		}
	}

	return patterns, nil
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
