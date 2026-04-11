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
