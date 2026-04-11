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
