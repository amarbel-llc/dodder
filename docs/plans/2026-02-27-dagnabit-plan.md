# dagnabit Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the bash `auto_reposition_packages` pipeline with a Go package and CLI tool that has pluggable interfaces for dependency reading, level mapping, and package moving.

**Architecture:** Single package `src/_/dagnabit` containing core graph types, three interfaces (DependencyReader, LevelMapper, PackageMover), a Repositioner orchestrator, dodder-specific implementations, and Kahn's topological sort. A thin CLI binary at `cmd/dagnabit/main.go` wires everything together. See `docs/plans/2026-02-27-dagnabit-design.md` for full design.

**Tech Stack:** Go stdlib, `os/exec` for shelling out to `go list` and `just`

**Skills:** @design_patterns-hamster_style, @design_patterns-snob_naming

---

### Task 1: Core types and interfaces

**Files:**
- Create: `go/src/_/dagnabit/main.go`

**Step 1: Write the types and interfaces file**

```go
package dagnabit

// Edge represents a directed dependency: Source depends on Target.
type Edge struct {
	Source string
	Target string
}

// DependencyReader produces directed edges from a codebase.
type DependencyReader interface {
	ReadDependencies() ([]Edge, error)
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
```

**Step 2: Verify it compiles**

Run: `cd go && go build ./src/_/dagnabit/`
Expected: success, no output

**Step 3: Commit**

```
git add go/src/_/dagnabit/main.go
git commit -m "feat(dagnabit): add core types and interfaces"
```

---

### Task 2: Topological sort with Kahn's algorithm

**Files:**
- Create: `go/src/_/dagnabit/topological_sort.go`
- Create: `go/src/_/dagnabit/topological_sort_test.go`

**Step 1: Write the failing tests**

```go
package dagnabit

import (
	"testing"
)

func TestTopologicalSortLinearChain(t *testing.T) {
	// a -> b -> c (a depends on b, b depends on c)
	edges := []Edge{
		{Source: "a", Target: "b"},
		{Source: "b", Target: "c"},
	}

	heights, err := TopologicalSort(edges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertHeight(t, heights, "c", 0)
	assertHeight(t, heights, "b", 1)
	assertHeight(t, heights, "a", 2)
}

func TestTopologicalSortDiamondDependency(t *testing.T) {
	// d depends on both b and c; both b and c depend on a
	edges := []Edge{
		{Source: "d", Target: "b"},
		{Source: "d", Target: "c"},
		{Source: "b", Target: "a"},
		{Source: "c", Target: "a"},
	}

	heights, err := TopologicalSort(edges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertHeight(t, heights, "a", 0)
	assertHeight(t, heights, "b", 1)
	assertHeight(t, heights, "c", 1)
	assertHeight(t, heights, "d", 2)
}

func TestTopologicalSortDisconnectedComponents(t *testing.T) {
	edges := []Edge{
		{Source: "a", Target: "b"},
		{Source: "c", Target: "d"},
	}

	heights, err := TopologicalSort(edges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertHeight(t, heights, "b", 0)
	assertHeight(t, heights, "a", 1)
	assertHeight(t, heights, "d", 0)
	assertHeight(t, heights, "c", 1)
}

func TestTopologicalSortCycleDetection(t *testing.T) {
	edges := []Edge{
		{Source: "a", Target: "b"},
		{Source: "b", Target: "a"},
	}

	_, err := TopologicalSort(edges)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

func TestTopologicalSortEmpty(t *testing.T) {
	heights, err := TopologicalSort(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(heights) != 0 {
		t.Fatalf("expected empty map, got %v", heights)
	}
}

func TestTopologicalSortSingleNode(t *testing.T) {
	edges := []Edge{
		{Source: "a", Target: "b"},
	}

	heights, err := TopologicalSort(edges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertHeight(t, heights, "b", 0)
	assertHeight(t, heights, "a", 1)
}

func assertHeight(t *testing.T, heights map[string]int, node string, expected int) {
	t.Helper()

	actual, ok := heights[node]
	if !ok {
		t.Errorf("node %q not found in heights", node)
		return
	}

	if actual != expected {
		t.Errorf("node %q: expected height %d, got %d", node, expected, actual)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd go && go test ./src/_/dagnabit/`
Expected: FAIL — `TopologicalSort` undefined

**Step 3: Write the implementation**

```go
package dagnabit

import "fmt"

// TopologicalSort performs Kahn's algorithm on the given edges and returns
// a map from node name to its height in the dependency DAG. Leaf nodes
// (no dependencies) have height 0. Returns an error if a cycle is detected.
func TopologicalSort(edges []Edge) (map[string]int, error) {
	if len(edges) == 0 {
		return make(map[string]int), nil
	}

	// Build adjacency list (reverse: dependency -> dependents) and in-degree map
	dependents := make(map[string][]string) // dep -> list of packages that depend on it
	inDegree := make(map[string]int)
	allNodes := make(map[string]struct{})

	for _, e := range edges {
		allNodes[e.Source] = struct{}{}
		allNodes[e.Target] = struct{}{}
		dependents[e.Target] = append(dependents[e.Target], e.Source)
		inDegree[e.Source]++

		if _, ok := inDegree[e.Target]; !ok {
			inDegree[e.Target] = 0
		}
	}

	// Initialize queue with zero in-degree nodes (leaves)
	queue := make([]string, 0)
	height := make(map[string]int)

	for node := range allNodes {
		height[node] = 0

		if inDegree[node] == 0 {
			queue = append(queue, node)
		}
	}

	sortedCount := 0

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sortedCount++

		for _, dependent := range dependents[current] {
			newHeight := height[current] + 1
			if newHeight > height[dependent] {
				height[dependent] = newHeight
			}

			inDegree[dependent]--

			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if sortedCount != len(allNodes) {
		return nil, fmt.Errorf("cycle detected in dependency graph")
	}

	return height, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd go && go test ./src/_/dagnabit/`
Expected: PASS

**Step 5: Commit**

```
git add go/src/_/dagnabit/topological_sort.go go/src/_/dagnabit/topological_sort_test.go
git commit -m "feat(dagnabit): add topological sort with Kahn's algorithm"
```

---

### Task 3: NATOLevelMapper

**Files:**
- Create: `go/src/_/dagnabit/nato_level_mapper.go`
- Create: `go/src/_/dagnabit/nato_level_mapper_test.go`

**Step 1: Write the failing tests**

```go
package dagnabit

import "testing"

func TestNATOLevelMapperHeight0(t *testing.T) {
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "_" {
		t.Errorf("expected %q, got %q", "_", name)
	}
}

func TestNATOLevelMapperHeight1(t *testing.T) {
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "alfa" {
		t.Errorf("expected %q, got %q", "alfa", name)
	}
}

func TestNATOLevelMapperMaxHeight(t *testing.T) {
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(26)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "zulu" {
		t.Errorf("expected %q, got %q", "zulu", name)
	}
}

func TestNATOLevelMapperOutOfRange(t *testing.T) {
	m := MakeNATOLevelMapper()

	_, err := m.LevelName(27)
	if err == nil {
		t.Fatal("expected error for out-of-range height, got nil")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd go && go test ./src/_/dagnabit/ -run TestNATO`
Expected: FAIL — `MakeNATOLevelMapper` undefined

**Step 3: Write the implementation**

```go
package dagnabit

import "fmt"

var natoLevels = []string{
	"_",
	"alfa",
	"bravo",
	"charlie",
	"delta",
	"echo",
	"foxtrot",
	"golf",
	"hotel",
	"india",
	"juliett",
	"kilo",
	"lima",
	"mike",
	"november",
	"oscar",
	"papa",
	"quebec",
	"romeo",
	"sierra",
	"tango",
	"uniform",
	"victor",
	"whiskey",
	"xray",
	"yankee",
	"zulu",
}

type NATOLevelMapper struct{}

func MakeNATOLevelMapper() NATOLevelMapper {
	return NATOLevelMapper{}
}

func (m NATOLevelMapper) LevelName(height int) (string, error) {
	if height < 0 || height >= len(natoLevels) {
		return "", fmt.Errorf(
			"height %d out of range (max %d)",
			height,
			len(natoLevels)-1,
		)
	}

	return natoLevels[height], nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd go && go test ./src/_/dagnabit/ -run TestNATO`
Expected: PASS

**Step 5: Commit**

```
git add go/src/_/dagnabit/nato_level_mapper.go go/src/_/dagnabit/nato_level_mapper_test.go
git commit -m "feat(dagnabit): add NATO phonetic alphabet level mapper"
```

---

### Task 4: GoListReader

**Files:**
- Create: `go/src/_/dagnabit/go_list_reader.go`

**Step 1: Write the implementation**

```go
package dagnabit

import (
	"bufio"
	"fmt"
	"os/exec"
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
	pattern := fmt.Sprintf("./%s/...", r.PackagePrefix)

	cmd := exec.Command(
		"go", "list",
		"-f", `{{.ImportPath}}{{"\t"}}{{range .Imports}}{{.}} {{end}}`,
		pattern,
	)
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

	return edges, scanner.Err()
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
```

**Step 2: Verify it compiles**

Run: `cd go && go build ./src/_/dagnabit/`
Expected: success

**Step 3: Commit**

```
git add go/src/_/dagnabit/go_list_reader.go
git commit -m "feat(dagnabit): add GoListReader using go list"
```

---

### Task 5: JustMover

**Files:**
- Create: `go/src/_/dagnabit/just_mover.go`

**Step 1: Write the implementation**

```go
package dagnabit

import (
	"fmt"
	"os"
	"os/exec"
)

// TODO: internalize the move logic (git mv + import rewriting + gofmt)
// instead of delegating to the justfile recipe.

// JustMover moves packages by shelling out to `just codemod-go-move_package`.
// Dir is the working directory containing the justfile.
type JustMover struct {
	Dir string
}

func (m JustMover) MovePackage(src, dst string) error {
	cmd := exec.Command("just", "codemod-go-move_package", src, dst)
	cmd.Dir = m.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("just codemod-go-move_package %s %s: %w", src, dst, err)
	}

	return nil
}
```

**Step 2: Verify it compiles**

Run: `cd go && go build ./src/_/dagnabit/`
Expected: success

**Step 3: Commit**

```
git add go/src/_/dagnabit/just_mover.go
git commit -m "feat(dagnabit): add JustMover delegating to justfile recipe"
```

---

### Task 6: Repositioner orchestrator

**Files:**
- Create: `go/src/_/dagnabit/repositioner.go`
- Create: `go/src/_/dagnabit/repositioner_test.go`

**Step 1: Write the failing tests**

```go
package dagnabit

import (
	"fmt"
	"strings"
	"testing"
)

type stubReader struct {
	edges []Edge
}

func (r stubReader) ReadDependencies() ([]Edge, error) {
	return r.edges, nil
}

type sliceLevelMapper struct {
	levels []string
}

func (m sliceLevelMapper) LevelName(height int) (string, error) {
	if height < 0 || height >= len(m.levels) {
		return "", fmt.Errorf("height %d out of range", height)
	}

	return m.levels[height], nil
}

type recordingMover struct {
	moves []string
}

func (m *recordingMover) MovePackage(src, dst string) error {
	m.moves = append(m.moves, fmt.Sprintf("%s -> %s", src, dst))
	return nil
}

func TestRepositionerMovesPackageToCorrectLevel(t *testing.T) {
	reader := stubReader{
		edges: []Edge{
			{Source: "level0/pkg_a", Target: "level0/pkg_b"},
		},
	}

	mapper := sliceLevelMapper{levels: []string{"level0", "level1"}}
	mover := &recordingMover{}

	r := Repositioner{
		Reader: reader,
		Mapper: mapper,
		Mover:  mover,
	}

	if err := r.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// pkg_a depends on pkg_b, so pkg_a should be at level1
	// pkg_b is already at level0 (correct), pkg_a needs to move from level0 to level1
	if len(mover.moves) != 1 {
		t.Fatalf("expected 1 move, got %d: %v", len(mover.moves), mover.moves)
	}

	expected := "src/level0/pkg_a -> src/level1/pkg_a"
	if mover.moves[0] != expected {
		t.Errorf("expected move %q, got %q", expected, mover.moves[0])
	}
}

func TestRepositionerSkipsCorrectlyPlacedPackages(t *testing.T) {
	reader := stubReader{
		edges: []Edge{
			{Source: "level1/pkg_a", Target: "level0/pkg_b"},
		},
	}

	mapper := sliceLevelMapper{levels: []string{"level0", "level1"}}
	mover := &recordingMover{}

	r := Repositioner{
		Reader: reader,
		Mapper: mapper,
		Mover:  mover,
	}

	if err := r.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mover.moves) != 0 {
		t.Fatalf("expected 0 moves, got %d: %v", len(mover.moves), mover.moves)
	}
}

func TestRepositionerDryRunDoesNotMove(t *testing.T) {
	reader := stubReader{
		edges: []Edge{
			{Source: "level0/pkg_a", Target: "level0/pkg_b"},
		},
	}

	mapper := sliceLevelMapper{levels: []string{"level0", "level1"}}
	mover := &recordingMover{}

	r := Repositioner{
		Reader: reader,
		Mapper: mapper,
		Mover:  mover,
		DryRun: true,
	}

	if err := r.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mover.moves) != 0 {
		t.Fatalf("expected 0 moves in dry run, got %d: %v", len(mover.moves), mover.moves)
	}
}

func TestRepositionerReaderError(t *testing.T) {
	reader := errorReader{err: fmt.Errorf("read failed")}
	mapper := sliceLevelMapper{levels: []string{"level0"}}
	mover := &recordingMover{}

	r := Repositioner{
		Reader: reader,
		Mapper: mapper,
		Mover:  mover,
	}

	err := r.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "read failed") {
		t.Errorf("expected error to contain %q, got %q", "read failed", err.Error())
	}
}

type errorReader struct {
	err error
}

func (r errorReader) ReadDependencies() ([]Edge, error) {
	return nil, r.err
}
```

**Step 2: Run tests to verify they fail**

Run: `cd go && go test ./src/_/dagnabit/ -run TestRepositioner`
Expected: FAIL — `Repositioner` has no `Run` method

**Step 3: Write the implementation**

```go
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
```

**Step 4: Run tests to verify they pass**

Run: `cd go && go test ./src/_/dagnabit/ -run TestRepositioner`
Expected: PASS

**Step 5: Run all dagnabit tests**

Run: `cd go && go test ./src/_/dagnabit/`
Expected: PASS (all tests)

**Step 6: Commit**

```
git add go/src/_/dagnabit/repositioner.go go/src/_/dagnabit/repositioner_test.go
git commit -m "feat(dagnabit): add Repositioner orchestrator"
```

---

### Task 7: CLI binary

**Files:**
- Create: `go/cmd/dagnabit/main.go`

**Step 1: Write the CLI binary**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/src/_/dagnabit"
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
			ModulePath:    "code.linenisgreat.com/dodder/go",
			Dir:           dir,
			PackagePrefix: "src",
		},
		Mapper: dagnabit.MakeNATOLevelMapper(),
		Mover:  dagnabit.JustMover{Dir: dir},
		DryRun: dryRun,
		Verbose: verbose,
	}

	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Verify it compiles (it will be built by `just build` alongside other binaries)**

Run: `cd go && go build ./cmd/dagnabit/`
Expected: success

**Step 3: Verify dry-run mode works against real codebase**

Run: `cd go && go run ./cmd/dagnabit/ -n`
Expected: either no output (everything placed correctly) or "would move:" lines

**Step 4: Commit**

```
git add go/cmd/dagnabit/main.go
git commit -m "feat(dagnabit): add CLI binary"
```

---

### Task 8: Dedup edges in GoListReader

The current `go list` approach will produce duplicate edges when multiple files
in the same package import the same dependency. The topological sort handles
duplicates correctly (they just add redundant in-degree counts that cancel out),
but deduplication is cleaner.

**Files:**
- Modify: `go/src/_/dagnabit/go_list_reader.go`

**Step 1: Add deduplication to ReadDependencies**

After the scanner loop, before returning, deduplicate edges:

```go
// After the scanner loop, deduplicate edges
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
```

**Step 2: Verify it compiles and tests pass**

Run: `cd go && go test ./src/_/dagnabit/`
Expected: PASS

**Step 3: Commit**

```
git add go/src/_/dagnabit/go_list_reader.go
git commit -m "fix(dagnabit): deduplicate edges in GoListReader"
```

---

### Task 9: End-to-end smoke test

**Files:**
- (none created — manual verification)

**Step 1: Build the full binary set**

Run: `cd go && just build`
Expected: builds all binaries including dagnabit

**Step 2: Run dagnabit in dry-run and compare to bash script**

Run both and compare output:

```bash
cd go
# New Go tool
./build/debug/dagnabit -n 2>/dev/null | sort > /tmp/dagnabit-go.txt

# Original bash pipeline
./bin/list_deps | ./bin/sort_packages | while IFS= read -r line; do
  if [[ $line =~ ^([0-9]+):\ (.+)$ ]]; then
    idx="${BASH_REMATCH[1]}"
    pkg="${BASH_REMATCH[2]}"
    levels=("_" "alfa" "bravo" "charlie" "delta" "echo" "foxtrot" "golf" "hotel" "india" "juliett" "kilo" "lima" "mike" "november" "oscar" "papa" "quebec" "romeo" "sierra" "tango" "uniform" "victor" "whiskey" "xray" "yankee" "zulu")
    required="${levels[$idx]}"
    current="${pkg%%/*}"
    if [[ "$required" != "$current" ]]; then
      name="${pkg#*/}"
      echo "would move: src/$pkg -> src/$required/$name"
    fi
  fi
done | sort > /tmp/dagnabit-bash.txt

diff /tmp/dagnabit-go.txt /tmp/dagnabit-bash.txt
```

Expected: identical output (or explainable differences due to edge dedup / ordering)

**Step 3: If outputs match, commit any final adjustments. If not, debug differences.**

---

### Task 10: Clean up old script (optional, confirm with user)

**Files:**
- Delete: `go/bin/auto_reposition_packages`

Only do this after confirming the Go tool produces equivalent output.

**Step 1: Confirm with user before deleting**

Ask whether to delete the old bash script or keep it as a reference.

**Step 2: If approved, delete and commit**

```
git rm go/bin/auto_reposition_packages
git commit -m "chore: remove bash auto_reposition_packages, replaced by dagnabit"
```
