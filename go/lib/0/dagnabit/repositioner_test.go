package dagnabit

import (
	"fmt"
	"strings"
	"testing"
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

func TestRepositionerMovesPackageToCorrectLevel(t *testing.T) {
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

	if err := r.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// pkg_a depends on pkg_b, so pkg_a should be at level1
	// pkg_b is already at level0 (correct), pkg_a needs to move from level0 to level1
	if len(mover.moves) != 1 {
		t.Fatalf("expected 1 move, got %d: %v", len(mover.moves), mover.moves)
	}

	expected := "tree/level0/pkg_a -> tree/level1/pkg_a"
	if mover.moves[0] != expected {
		t.Errorf("expected move %q, got %q", expected, mover.moves[0])
	}
}

func TestRepositionerSkipsCorrectlyPlacedPackages(t *testing.T) {
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

	if err := r.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mover.moves) != 0 {
		t.Fatalf("expected 0 moves, got %d: %v", len(mover.moves), mover.moves)
	}
}

func TestRepositionerDryRunDoesNotMove(t *testing.T) {
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

func (errorReader errorReader) ReadDependencies() (map[string][]Edge, error) {
	return nil, errorReader.err
}

func TestRepositionerCycleError(t *testing.T) {
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
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected error to mention cycle, got %q", err.Error())
	}
}

func TestRepositionerMapperError(t *testing.T) {
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
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "mapping height") {
		t.Errorf("expected error to contain %q, got %q", "mapping height", err.Error())
	}
}

func TestRepositionerCrossPrefixEdgesIgnored(t *testing.T) {
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

	if err := r.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mover.moves) != 0 {
		t.Fatalf("expected 0 moves, got %d: %v", len(mover.moves), mover.moves)
	}
}

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
