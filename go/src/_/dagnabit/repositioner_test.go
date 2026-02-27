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
