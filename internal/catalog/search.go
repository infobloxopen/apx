package catalog

import (
	"strings"
)

// SearchOptions holds filter criteria for catalog search.
type SearchOptions struct {
	Query     string // free-text match on ID/Name/Description
	Format    string // filter by schema format
	Lifecycle string // filter by lifecycle (experimental, beta, stable, deprecated, sunset)
	Domain    string // filter by domain (e.g. "payments")
	APILine   string // filter by API line (e.g. "v1")
	Origin    string // filter by origin: "first-party", "external", "forked", or "" (all)
}

// SearchModules searches the catalog for modules matching the query and format.
// This is the legacy two-argument form; prefer SearchModulesOpts for new code.
func SearchModules(gen *Generator, query, format string) ([]Module, error) {
	return SearchModulesOpts(gen, SearchOptions{
		Query:  query,
		Format: format,
	})
}

// SearchModulesOpts searches the catalog with full filter support.
func SearchModulesOpts(gen *Generator, opts SearchOptions) ([]Module, error) {
	catalog, err := gen.Load()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(opts.Query)
	matches := []Module{}

	for _, module := range catalog.Modules {
		// Filter by origin
		if opts.Origin != "" {
			switch opts.Origin {
			case "first-party":
				if module.Origin != "" {
					continue
				}
			case "external":
				if module.Origin != "external" {
					continue
				}
			case "forked":
				if module.Origin != "forked" {
					continue
				}
			}
		}

		// Filter by format
		if opts.Format != "" && !strings.EqualFold(module.Format, opts.Format) {
			continue
		}

		// Filter by lifecycle
		if opts.Lifecycle != "" && !strings.EqualFold(module.Lifecycle, opts.Lifecycle) {
			continue
		}

		// Filter by domain
		if opts.Domain != "" && !strings.EqualFold(module.Domain, opts.Domain) {
			continue
		}

		// Filter by API line
		if opts.APILine != "" && !strings.EqualFold(module.APILine, opts.APILine) {
			continue
		}

		// Filter by free-text query
		if opts.Query != "" {
			display := strings.ToLower(module.DisplayName())
			desc := strings.ToLower(module.Description)
			domain := strings.ToLower(module.Domain)

			if !strings.Contains(display, queryLower) &&
				!strings.Contains(desc, queryLower) &&
				!strings.Contains(domain, queryLower) {
				continue
			}
		}

		matches = append(matches, module)
	}

	return matches, nil
}
