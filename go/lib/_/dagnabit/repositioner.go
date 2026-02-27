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

func (r *Repositioner) Run() error {
	edges, err := r.Reader.ReadDependencies()
	if err != nil {
		return fmt.Errorf("reading dependencies: %w", err)
	}

	heights, err := TopologicalSort(edges)
	if err != nil {
		return fmt.Errorf("topological sort: %w", err)
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
		requiredLevel, err := r.Mapper.LevelName(height)
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

		if r.DryRun {
			fmt.Printf("would move: %s -> %s\n", srcPath, dstPath)
			continue
		}

		if err := r.Mover.MovePackage(srcPath, dstPath); err != nil {
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
