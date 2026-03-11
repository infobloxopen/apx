package catalog

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/pkg/githubauth"
)

// ghPackage represents a minimal GitHub Package from the Packages API.
type ghPackage struct {
	Name string `json:"name"`
}

// DiscoverRegistries queries the GitHub Packages API for catalog containers
// in the given org. It returns a RegistrySource for each package whose name
// ends with "-catalog".
//
// Uses the githubauth package for authentication. Returns nil (no sources)
// if authentication is unavailable or no catalogs are found.
func DiscoverRegistries(org string) []CatalogSource {
	return discoverRegistries(org, ghRunDiscover)
}

// ghRunDiscover is the default implementation that calls the GitHub API
// via githubauth.
var ghRunDiscover = ghRunDiscoverReal

func ghRunDiscoverReal(org string) ([]byte, error) {
	token, err := githubauth.EnsureToken(org)
	if err != nil {
		return nil, fmt.Errorf("GitHub auth failed: %w", err)
	}

	client := githubauth.NewClient(token)
	items, err := client.GetPaginated(fmt.Sprintf("/orgs/%s/packages?package_type=container", org))
	if err != nil {
		return nil, err
	}

	// Re-serialize the items as a JSON array for backward compatibility
	// with the existing parser below.
	return json.Marshal(items)
}

// discoverRegistries is the testable core — accepts a runner function.
func discoverRegistries(org string, runner func(string) ([]byte, error)) []CatalogSource {
	output, err := runner(org)
	if err != nil {
		return nil // auth not available or API error — silent fallback
	}

	var packages []ghPackage
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil
	}

	var sources []CatalogSource
	for _, pkg := range packages {
		if !strings.HasSuffix(pkg.Name, CatalogImageSuffix) {
			continue
		}
		// Derive the repo name from the package name by stripping the suffix
		repoName := strings.TrimSuffix(pkg.Name, CatalogImageSuffix)
		if repoName == "" {
			continue
		}

		src := &RegistrySource{
			Org:  org,
			Repo: repoName,
		}
		sources = append(sources, &CachedSource{
			Inner:    src,
			CacheDir: DefaultCacheDir(org, repoName),
		})
	}
	return sources
}
