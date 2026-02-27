# dagnabit: Reusable DAG-based package repositioner

## Problem

The `go/bin/auto_reposition_packages` bash script automatically moves Go
packages to correct NATO phonetic levels based on their dependency graph. It
works, but is bash-only, tightly coupled to dodder conventions, and not reusable
across other codebases that want similar DAG-based directory organization.

## Goal

A Go package and CLI tool that:

- Reads dependencies from a codebase (pluggable)
- Topologically sorts them and assigns level heights
- Maps heights to named levels (pluggable)
- Moves misplaced packages to their correct level (pluggable)
- Works for dodder's NATO hierarchy today, other codebases tomorrow

## Package location

- **Library**: `src/_/dagnabit` (no internal dodder dependencies)
- **CLI binary**: `cmd/dagnabit/main.go`

## Core types

```go
package dagnabit

// Edge represents a directed dependency: Source depends on Target.
type Edge struct {
    Source string
    Target string
}
```

## Interfaces

```go
// DependencyReader produces directed edges from a codebase.
type DependencyReader interface {
    ReadDependencies() ([]Edge, error)
}

// LevelMapper assigns names to topological heights.
type LevelMapper interface {
    LevelName(height int) (string, error)
}

// PackageMover executes the move of a package from one path to another.
type PackageMover interface {
    MovePackage(src, dst string) error
}
```

## Orchestrator

```go
type Repositioner struct {
    Reader  DependencyReader
    Mapper  LevelMapper
    Mover   PackageMover
    DryRun  bool
    Verbose bool
}

func (r *Repositioner) Run() error
```

`Run()` does:

1. `Reader.ReadDependencies()` to get edges
2. Build directed graph, run Kahn's topological sort, compute heights
3. For each node, compare `Mapper.LevelName(height)` to the node's current
   level prefix (the part before the first `/`)
4. If they differ, call `Mover.MovePackage(currentPath, newPath)` (skipped
   when `DryRun` is true)
5. Stop on first move error

## Dodder-specific implementations

All live in `src/_/dagnabit` alongside the interfaces.

### GoListReader

Shells out to `go list -f '{{range .Imports}}...'` via `os/exec`. Filters
imports to those matching the module path, strips the module prefix, produces
`[]Edge`.

TODO comment: refactor to `golang.org/x/tools/go/packages` for direct
programmatic access.

### NATOLevelMapper

Ordered slice of NATO level names: `"_", "alfa", "bravo", ..., "zulu"`.
Returns the name at the given height index. Errors if height exceeds available
levels.

### JustMover

Shells out to `just codemod-go-move_package src dst` via `os/exec`.

TODO comment: internalize the move logic (git mv + import rewriting + gofmt).

## Graph algorithm

Kahn's algorithm (matching the existing bash `sort_packages` implementation):

1. Build adjacency list and in-degree map from edges
2. Initialize queue with zero in-degree nodes
3. Process queue: for each node, height = `max(dependency_heights) + 1`
4. If sorted count < total nodes, return error (cycle detected)

## CLI interface

```
dagnabit [OPTIONS]

Options:
    -n, --dry-run     Show what would be moved without moving
    -v, --verbose     Enable verbose output
    -h, --help        Show help
```

Must be run from a directory containing `go.mod` (matching current behavior).

## Testing

- **Graph/sort unit tests**: Known edge sets, assert correct heights and order.
  Cycle detection test.
- **NATOLevelMapper unit tests**: Height 0, max height, out-of-range.
- **GoListReader integration test**: Run against actual dodder module.
- **JustMover**: Tested implicitly via existing justfile recipe.

## Extraction path

The `src/_/dagnabit` package has zero dodder-internal dependencies. When
extracting to a standalone repo:

1. Move `src/_/dagnabit` to a new module
2. Move dodder-specific implementations (GoListReader, NATOLevelMapper,
   JustMover) into the dodder `cmd/dagnabit` binary
3. Import the library for interfaces + graph logic
