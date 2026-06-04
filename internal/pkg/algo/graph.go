package algo

import "maps"

func PruneTree[T comparable](rootID T, parent map[T]T) map[T]T {
	// Copy to avoid mutating the original map.
	p := make(map[T]T, len(parent))
	maps.Copy(p, parent)
	// Root must not have a parent.
	delete(p, rootID)
	// parent -> children
	children := make(map[T][]T)
	for child, par := range p {
		children[par] = append(children[par], child)
	}

	result := make(map[T]T, 0)
	visited := make(map[T]struct{})

	var dfs func(T)
	dfs = func(node T) {
		if _, ok := visited[node]; ok {
			return
		}

		visited[node] = struct{}{}

		for _, child := range children[node] {
			result[child] = node
			dfs(child)
		}
	}

	dfs(rootID)

	return result
}
