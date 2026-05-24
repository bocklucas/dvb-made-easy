package restore

import "sort"

func TopoSort(services []string, deps map[string][]string) []string {
	sorted := make([]string, 0, len(services))
	visited := make(map[string]bool)

	serviceSet := make(map[string]bool, len(services))
	for _, s := range services {
		serviceSet[s] = true
	}

	alphabetical := make([]string, len(services))
	copy(alphabetical, services)
	sort.Strings(alphabetical)

	var visit func(string)
	visit = func(s string) {
		if visited[s] {
			return
		}
		visited[s] = true
		for _, dep := range deps[s] {
			if serviceSet[dep] {
				visit(dep)
			}
		}
		sorted = append(sorted, s)
	}

	for _, s := range alphabetical {
		visit(s)
	}

	return sorted
}

func ReverseOrder(order []string) []string {
	reversed := make([]string, len(order))
	for i, s := range order {
		reversed[len(order)-1-i] = s
	}
	return reversed
}
