package dagnabit

// Edge represents a directed dependency: Source depends on Target.
type Edge struct {
	Source string
	Target string
}

// DependencyReader produces directed edges from a codebase,
// grouped by tree prefix (e.g., "lib", "internal").
type DependencyReader interface {
	ReadDependencies() (map[string][]Edge, error)
}

// LevelMapper assigns names to topological heights.
// Height 0 is the lowest level (no internal dependencies).
type LevelMapper interface {
	LevelName(height int) (string, error)
}

// PackageMover executes the move of a package from one path to another.
type PackageMover interface {
	MovePackage(src, dst string) error
}
