package dagnabit

import (
	"fmt"
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
		return err
	}

	for node, height := range heights {
		requiredLevel, err := r.Mapper.LevelName(height)
		if err != nil {
			return fmt.Errorf("mapping height %d for %s: %w", height, node, err)
		}

		currentLevel := extractLevel(node)

		if currentLevel == requiredLevel {
			continue
		}

		packageName := extractPackageName(node)
		srcPath := "src/" + node
		dstPath := "src/" + requiredLevel + "/" + packageName

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

// extractLevel returns the part before the first "/" in a package path.
func extractLevel(path string) string {
	if idx := strings.IndexByte(path, '/'); idx >= 0 {
		return path[:idx]
	}

	return path
}

// extractPackageName returns the part after the first "/" in a package path.
func extractPackageName(path string) string {
	if idx := strings.IndexByte(path, '/'); idx >= 0 {
		return path[idx+1:]
	}

	return path
}
