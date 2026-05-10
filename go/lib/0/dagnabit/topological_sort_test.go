package dagnabit

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestTopologicalSortLinearChain(t1 *testing.T) {
	t := ui.MakeT(t1)
	// a -> b -> c (a depends on b, b depends on c)
	edges := []Edge{
		{Source: "a", Target: "b"},
		{Source: "b", Target: "c"},
	}

	heights, err := TopologicalSort(edges)
	t.AssertNoError(err)

	assertHeight(&t, heights, "c", 0)
	assertHeight(&t, heights, "b", 1)
	assertHeight(&t, heights, "a", 2)
}

func TestTopologicalSortDiamondDependency(t1 *testing.T) {
	t := ui.MakeT(t1)
	// d depends on both b and c; both b and c depend on a
	edges := []Edge{
		{Source: "d", Target: "b"},
		{Source: "d", Target: "c"},
		{Source: "b", Target: "a"},
		{Source: "c", Target: "a"},
	}

	heights, err := TopologicalSort(edges)
	t.AssertNoError(err)

	assertHeight(&t, heights, "a", 0)
	assertHeight(&t, heights, "b", 1)
	assertHeight(&t, heights, "c", 1)
	assertHeight(&t, heights, "d", 2)
}

func TestTopologicalSortDisconnectedComponents(t1 *testing.T) {
	t := ui.MakeT(t1)
	edges := []Edge{
		{Source: "a", Target: "b"},
		{Source: "c", Target: "d"},
	}

	heights, err := TopologicalSort(edges)
	t.AssertNoError(err)

	assertHeight(&t, heights, "b", 0)
	assertHeight(&t, heights, "a", 1)
	assertHeight(&t, heights, "d", 0)
	assertHeight(&t, heights, "c", 1)
}

func TestTopologicalSortCycleDetection(t1 *testing.T) {
	t := ui.MakeT(t1)
	edges := []Edge{
		{Source: "a", Target: "b"},
		{Source: "b", Target: "a"},
	}

	_, err := TopologicalSort(edges)
	t.AssertError(err)
}

func TestTopologicalSortEmpty(t1 *testing.T) {
	t := ui.MakeT(t1)
	heights, err := TopologicalSort(nil)
	t.AssertNoError(err)

	t.AssertLen(0, heights, "heights")
}

func TestTopologicalSortSingleNode(t1 *testing.T) {
	t := ui.MakeT(t1)
	edges := []Edge{
		{Source: "a", Target: "b"},
	}

	heights, err := TopologicalSort(edges)
	t.AssertNoError(err)

	assertHeight(&t, heights, "b", 0)
	assertHeight(&t, heights, "a", 1)
}

func assertHeight(t *ui.T, heights map[string]int, node string, expected int) {
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
