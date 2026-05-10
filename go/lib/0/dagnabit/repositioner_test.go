package dagnabit

import (
	"fmt"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

type stubReader struct {
	edgesByPrefix map[string][]Edge
}

func (stubReader stubReader) ReadDependencies() (map[string][]Edge, error) {
	return stubReader.edgesByPrefix, nil
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

func TestRepositionerMovesPackageToCorrectLevel(t1 *testing.T) {
	t := ui.MakeT(t1)
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level0/pkg_a", Target: "tree/level0/pkg_b"},
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

	t.AssertNoError(r.Run())

	// pkg_a depends on pkg_b, so pkg_a should be at level1
	// pkg_b is already at level0 (correct), pkg_a needs to move from level0 to level1
	if len(mover.moves) != 1 {
		t.Fatalf("expected 1 move, got %d: %v", len(mover.moves), mover.moves)
	}

	expected := "tree/level0/pkg_a -> tree/level1/pkg_a"
	t.AssertEqualStrings(expected, mover.moves[0])
}

func TestRepositionerSkipsCorrectlyPlacedPackages(t1 *testing.T) {
	t := ui.MakeT(t1)
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level1/pkg_a", Target: "tree/level0/pkg_b"},
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

	t.AssertNoError(r.Run())

	if len(mover.moves) != 0 {
		t.Fatalf("expected 0 moves, got %d: %v", len(mover.moves), mover.moves)
	}
}

func TestRepositionerDryRunDoesNotMove(t1 *testing.T) {
	t := ui.MakeT(t1)
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level0/pkg_a", Target: "tree/level0/pkg_b"},
			},
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

	t.AssertNoError(r.Run())

	if len(mover.moves) != 0 {
		t.Fatalf("expected 0 moves in dry run, got %d: %v", len(mover.moves), mover.moves)
	}
}

func TestRepositionerReaderError(t1 *testing.T) {
	t := ui.MakeT(t1)
	reader := errorReader{err: fmt.Errorf("read failed")}
	mapper := sliceLevelMapper{levels: []string{"level0"}}
	mover := &recordingMover{}

	r := Repositioner{
		Reader: reader,
		Mapper: mapper,
		Mover:  mover,
	}

	err := r.Run()
	t.AssertErrorContains("read failed", err)
}

type errorReader struct {
	err error
}

func (errorReader errorReader) ReadDependencies() (map[string][]Edge, error) {
	return nil, errorReader.err
}

func TestRepositionerCycleError(t1 *testing.T) {
	t := ui.MakeT(t1)
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level0/pkg_a", Target: "tree/level0/pkg_b"},
				{Source: "tree/level0/pkg_b", Target: "tree/level0/pkg_a"},
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

	err := r.Run()
	t.AssertErrorContains("cycle", err)
}

func TestRepositionerMapperError(t1 *testing.T) {
	t := ui.MakeT(t1)
	// With only 1 level defined, height 1 is out of range
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"tree": {
				{Source: "tree/level0/pkg_a", Target: "tree/level0/pkg_b"},
			},
		},
	}

	mapper := sliceLevelMapper{levels: []string{"level0"}}
	mover := &recordingMover{}

	r := Repositioner{
		Reader: reader,
		Mapper: mapper,
		Mover:  mover,
	}

	err := r.Run()
	t.AssertErrorContains("mapping height", err)
}

func TestRepositionerCrossPrefixEdgesIgnored(t1 *testing.T) {
	t := ui.MakeT(t1)
	reader := stubReader{
		edgesByPrefix: map[string][]Edge{
			"treeA": {},
			"treeB": {},
		},
	}

	mapper := sliceLevelMapper{levels: []string{"level0", "level1"}}
	mover := &recordingMover{}

	r := Repositioner{
		Reader: reader,
		Mapper: mapper,
		Mover:  mover,
	}

	t.AssertNoError(r.Run())

	if len(mover.moves) != 0 {
		t.Fatalf("expected 0 moves, got %d: %v", len(mover.moves), mover.moves)
	}
}

func TestRepositionerMultiplePrefixesSortedIndependently(t1 *testing.T) {
	t := ui.MakeT(t1)
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

	t.AssertNoError(r.Run())

	// Both pkg_a and pkg_c should move from level0 to level1
	if len(mover.moves) != 2 {
		t.Fatalf("expected 2 moves, got %d: %v", len(mover.moves), mover.moves)
	}

	// Sorted by prefix: internal first, then lib
	expected0 := "internal/level0/pkg_c -> internal/level1/pkg_c"
	expected1 := "lib/level0/pkg_a -> lib/level1/pkg_a"

	t.AssertEqualStrings(expected0, mover.moves[0])
	t.AssertEqualStrings(expected1, mover.moves[1])
}
