package dagnabit

import (
	"fmt"
	"sort"
	"strings"
)

// Repositioner orchestrates dependency reading, topological sorting,
// level mapping, and package moving.
type Repositioner struct {
	Reader  DependencyReader
	Mapper  LevelMapper
	Mover   PackageMover
	DryRun  bool
	Verbose bool
}

func (repositioner *Repositioner) Run() error {
	edgesByPrefix, err := repositioner.Reader.ReadDependencies()
	if err != nil {
		return fmt.Errorf("reading dependencies: %w", err)
	}

	prefixes := make([]string, 0, len(edgesByPrefix))
	for prefix := range edgesByPrefix {
		prefixes = append(prefixes, prefix)
	}

	sort.Strings(prefixes)

	for _, prefix := range prefixes {
		if err := repositioner.runPrefix(prefix, edgesByPrefix[prefix]); err != nil {
			return err
		}
	}

	return nil
}

func (repositioner *Repositioner) runPrefix(prefix string, edges []Edge) error {
	heights, err := TopologicalSort(edges)
	if err != nil {
		return fmt.Errorf("topological sort for %s: %w", prefix, err)
	}

	type nodeHeight struct {
		node   string
		height int
	}

	sorted := make([]nodeHeight, 0, len(heights))
	for node, height := range heights {
		sorted = append(sorted, nodeHeight{node: node, height: height})
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].height != sorted[j].height {
			return sorted[i].height < sorted[j].height
		}
		return sorted[i].node < sorted[j].node
	})

	for _, nh := range sorted {
		node := nh.node
		height := nh.height
		requiredLevel, err := repositioner.Mapper.LevelName(height)
		if err != nil {
			return fmt.Errorf("mapping height %d for %s: %w", height, node, err)
		}

		currentLevel := extractLevel(node)

		if currentLevel == requiredLevel {
			continue
		}

		treePrefix := extractTreePrefix(node)
		packageName := extractPackageName(node)
		srcPath := node
		dstPath := treePrefix + "/" + requiredLevel + "/" + packageName

		if repositioner.DryRun {
			fmt.Printf("would move: %s -> %s\n", srcPath, dstPath)
			continue
		}

		if err := repositioner.Mover.MovePackage(srcPath, dstPath); err != nil {
			return fmt.Errorf("moving %s to %s: %w", srcPath, dstPath, err)
		}
	}

	return nil
}

// extractTreePrefix returns the first path component (e.g., "lib" from "lib/_/ohio_buffer").
func extractTreePrefix(path string) string {
	if idx := strings.IndexByte(path, '/'); idx >= 0 {
		return path[:idx]
	}

	return path
}

// extractLevel returns the second path component (e.g., "_" from "lib/_/ohio_buffer").
func extractLevel(path string) string {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return path
	}

	return parts[1]
}

// extractPackageName returns the third path component (e.g., "ohio_buffer" from "lib/_/ohio_buffer").
func extractPackageName(path string) string {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 3 {
		return path
	}

	return parts[2]
}
