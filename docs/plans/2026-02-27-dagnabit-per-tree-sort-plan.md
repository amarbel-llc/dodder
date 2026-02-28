# dagnabit per-tree sorting Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Change dagnabit to sort each tree prefix (lib/, internal/) independently so that gaps in individual trees are eliminated.

**Architecture:** `DependencyReader` returns `map[string][]Edge` keyed by tree prefix. `GoListReader.readPrefix` filters out cross-prefix edges. `Repositioner.Run` iterates per prefix, running `TopologicalSort` and level mapping independently for each tree. No changes to `TopologicalSort` or `NATOLevelMapper`.

**Tech Stack:** Go stdlib, existing dagnabit package

**Skills:** @design_patterns-hamster_style

---

### Task 1: Change DependencyReader interface to return per-prefix edges

**Files:**
- Modify: `go/lib/_/dagnabit/main.go`

**Step 1: Update the interface**

Change `DependencyReader` to return `map[string][]Edge`:

```go
// DependencyReader produces directed edges from a codebase,
// grouped by tree prefix (e.g., "lib", "internal").
type DependencyReader interface {
	ReadDependencies() (map[string][]Edge, error)
}
```

**Step 2: Verify it compiles (expect failures)**

Run: `cd go && go build ./lib/_/dagnabit/`
Expected: FAIL — `GoListReader` and test stubs return `[]Edge`, not `map[string][]Edge`

**Step 3: Commit**

```
git add go/lib/_/dagnabit/main.go
git commit -m "refactor(dagnabit): change DependencyReader to return per-prefix edges"
```

---

### Task 2: Update GoListReader to return per-prefix edges and filter cross-prefix deps

**Files:**
- Modify: `go/lib/_/dagnabit/go_list_reader.go`

**Step 1: Update ReadDependencies to return map and filter cross-prefix edges**

Replace the `ReadDependencies` method:

```go
func (goListReader GoListReader) ReadDependencies() (map[string][]Edge, error) {
	result := make(map[string][]Edge)

	for _, prefix := range goListReader.PackagePrefixes {
		edges, err := goListReader.readPrefix(prefix)
		if err != nil {
			return nil, err
		}

		result[prefix] = edges
	}

	return result, nil
}
```

**Step 2: Filter cross-prefix edges in readPrefix**

In `readPrefix`, edges already have the prefix as the first path component
(e.g., `"internal/alfa/domain_interfaces"`). Currently all imports matching
`modulePrefix` are included. Add a filter so only imports starting with
`modulePrefix + prefix + "/"` are kept as targets.

Replace the inner import filtering loop (lines 103–116) so cross-prefix imports
are excluded:

```go
		prefixFilter := modulePrefix + prefix + "/"

		imports := strings.Fields(parts[1])

		for _, imp := range imports {
			if !strings.HasPrefix(imp, prefixFilter) {
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
```

Also update the source filter (line 89) to use `prefixFilter` instead of
`modulePrefix`:

```go
		if !strings.HasPrefix(sourceFull, prefixFilter) {
			continue
		}
```

And update the dedup to happen per-prefix within `readPrefix` (move the
dedup block from `ReadDependencies` into `readPrefix`, before the return):

```go
	// Deduplicate edges
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

**Step 3: Verify it compiles**

Run: `cd go && go build ./lib/_/dagnabit/`
Expected: FAIL — `Repositioner` and test stubs still use old interface

---

### Task 3: Update Repositioner to iterate per prefix

**Files:**
- Modify: `go/lib/_/dagnabit/repositioner.go`

**Step 1: Update Run to iterate per prefix**

Replace the `Run` method:

```go
func (repositioner *Repositioner) Run() error {
	edgesByPrefix, err := repositioner.Reader.ReadDependencies()
	if err != nil {
		return fmt.Errorf("reading dependencies: %w", err)
	}

	// Sort prefix keys for deterministic ordering
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
```

**Step 2: Verify it compiles**

Run: `cd go && go build ./lib/_/dagnabit/`
Expected: FAIL — test stubs still use old interface

---

### Task 4: Update test stubs and existing tests

**Files:**
- Modify: `go/lib/_/dagnabit/repositioner_test.go`

**Step 1: Update stubReader to return map**

```go
type stubReader struct {
	edgesByPrefix map[string][]Edge
}

func (stubReader stubReader) ReadDependencies() (map[string][]Edge, error) {
	return stubReader.edgesByPrefix, nil
}
```

**Step 2: Update errorReader to return map**

```go
func (errorReader errorReader) ReadDependencies() (map[string][]Edge, error) {
	return nil, errorReader.err
}
```

**Step 3: Update all existing tests to use the new stub format**

Each test that previously created `stubReader{edges: []Edge{...}}` needs to
wrap edges in a map. The tree prefix is `"tree"` (matching the existing node
format `"tree/level0/pkg_a"`):

For `TestRepositionerMovesPackageToCorrectLevel`:
```go
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level0/pkg_a", Target: "tree/level0/pkg_b"},
			},
		},
	}
```

For `TestRepositionerSkipsCorrectlyPlacedPackages`:
```go
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level1/pkg_a", Target: "tree/level0/pkg_b"},
			},
		},
	}
```

For `TestRepositionerDryRunDoesNotMove`:
```go
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level0/pkg_a", Target: "tree/level0/pkg_b"},
			},
		},
	}
```

For `TestRepositionerCycleError`:
```go
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level0/pkg_a", Target: "tree/level0/pkg_b"},
				{Source: "tree/level0/pkg_b", Target: "tree/level0/pkg_a"},
			},
		},
	}
```

For `TestRepositionerMapperError`:
```go
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level0/pkg_a", Target: "tree/level0/pkg_b"},
			},
		},
	}
```

**Step 4: Verify all existing tests pass**

Run: `cd go && go test ./lib/_/dagnabit/`
Expected: PASS

**Step 5: Commit**

```
git add go/lib/_/dagnabit/main.go go/lib/_/dagnabit/go_list_reader.go go/lib/_/dagnabit/repositioner.go go/lib/_/dagnabit/repositioner_test.go
git commit -m "refactor(dagnabit): sort each tree prefix independently

DependencyReader now returns map[string][]Edge keyed by tree prefix.
GoListReader filters out cross-prefix edges so each tree is sorted
in isolation. This closes gaps when a level (e.g., bravo) has no
packages in a particular tree."
```

---

### Task 5: Add test for cross-prefix edge filtering

**Files:**
- Modify: `go/lib/_/dagnabit/repositioner_test.go`

**Step 1: Write the test**

This test verifies that packages in two different tree prefixes are sorted
independently. `treeA/level1/pkg_x` depends on `treeB/level0/pkg_y` in the
real codebase, but in the per-tree model that cross-prefix edge is invisible.
Both packages should be placed at level0 within their respective trees.

```go
func TestRepositionerCrossPrefixEdgesIgnored(t *testing.T) {
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"treeA": {
				// pkg_x has no intra-treeA deps, so height 0
			},
			"treeB": {
				// pkg_y has no intra-treeB deps, so height 0
			},
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

	// No nodes appear in edges, so nothing to move
	if len(mover.moves) != 0 {
		t.Fatalf("expected 0 moves, got %d: %v", len(mover.moves), mover.moves)
	}
}
```

**Step 2: Write a test with two independent tree prefixes that each have moves**

```go
func TestRepositionerMultiplePrefixesSortedIndependently(t *testing.T) {
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"lib": {
				{Source: "lib/level0/pkg_a", Target: "lib/level0/pkg_b"},
			},
			"internal": {
				{Source: "internal/level0/pkg_c", Target: "internal/level0/pkg_d"},
			},
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

	// Both pkg_a and pkg_c should move from level0 to level1
	if len(mover.moves) != 2 {
		t.Fatalf("expected 2 moves, got %d: %v", len(mover.moves), mover.moves)
	}

	// Sorted by prefix: internal first, then lib
	expected0 := "internal/level0/pkg_c -> internal/level1/pkg_c"
	expected1 := "lib/level0/pkg_a -> lib/level1/pkg_a"

	if mover.moves[0] != expected0 {
		t.Errorf("move[0]: expected %q, got %q", expected0, mover.moves[0])
	}

	if mover.moves[1] != expected1 {
		t.Errorf("move[1]: expected %q, got %q", expected1, mover.moves[1])
	}
}
```

**Step 3: Run tests**

Run: `cd go && go test ./lib/_/dagnabit/`
Expected: PASS

**Step 4: Commit**

```
git add go/lib/_/dagnabit/repositioner_test.go
git commit -m "test(dagnabit): add tests for per-prefix independent sorting"
```

---

### Task 6: Update CLI binary

**Files:**
- Verify: `go/cmd/dagnabit/main.go`

**Step 1: Verify the CLI still compiles**

The CLI uses `GoListReader` which now returns `map[string][]Edge`. Since it
implements the updated `DependencyReader` interface, no CLI changes should be
needed.

Run: `cd go && go build ./cmd/dagnabit/`
Expected: PASS

**Step 2: Run dry-run to verify output**

Run: `cd go && go run ./cmd/dagnabit/ -n 2>&1 | head -20`
Expected: "would move:" lines showing internal/ packages shifting down

**Step 3: Verify the full dry-run output is reasonable**

Run: `cd go && go run ./cmd/dagnabit/ -n 2>&1 | wc -l`
Expected: ~87 lines (one per internal/ package that needs to move, plus any
lib/ packages — lib/ should have zero moves)

Run: `cd go && go run ./cmd/dagnabit/ -n 2>&1 | grep '^would move: lib/'`
Expected: no output (lib/ packages should already be correctly placed)

**Step 4: Commit if any CLI changes were needed**

(Likely no commit needed for this task.)

---

### Task 7: Run full test suite

**Files:**
- (none — verification only)

**Step 1: Run dagnabit unit tests**

Run: `cd go && go test -v -tags test,debug ./lib/_/dagnabit/`
Expected: all PASS

**Step 2: Run full Go unit tests**

Run: `cd go && just test-go`
Expected: PASS (dagnabit changes don't affect other packages)

**Step 3: Commit design doc**

```
git add docs/plans/2026-02-27-dagnabit-per-tree-sort-design.md docs/plans/2026-02-27-dagnabit-per-tree-sort-plan.md
git commit -m "docs: add per-tree sort design and plan for dagnabit"
```
