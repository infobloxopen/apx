package catalog

import (
	"strings"
)

// SearchModules searches the catalog for modules matching the query and format
func SearchModules(gen *Generator, query, format string) ([]Module, error) {
	catalog, err := gen.Load()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	matches := []Module{}

	for _, module := range catalog.Modules {
		// Filter by format if specified
		if format != "" && strings.ToLower(module.Format) != strings.ToLower(format) {
			continue
		}

		// Filter by query if specified
		if query != "" {
			nameMatch := strings.Contains(strings.ToLower(module.Name), queryLower)
			descMatch := strings.Contains(strings.ToLower(module.Description), queryLower)

			if !nameMatch && !descMatch {
				continue
			}
		}

		matches = append(matches, module)
	}

	return matches, nil
}
