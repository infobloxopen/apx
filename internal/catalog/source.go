package catalog

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// CatalogSource provides a Catalog from some origin.
type CatalogSource interface {
	// Load returns the catalog from this source.
	// Implementations should return a non-nil error only for unexpected
	// failures. A missing or empty catalog is not an error — return an
	// empty Catalog with Version=1 instead.
	Load() (*Catalog, error)

	// Name returns a human-readable identifier for logging and errors.
	Name() string
}

// ---------------------------------------------------------------------------
// LocalSource — reads catalog.yaml from the local filesystem.
// ---------------------------------------------------------------------------

// LocalSource reads a catalog from a local YAML file.
type LocalSource struct {
	Path string
}

// Load reads the catalog from the local filesystem.
// Returns an empty catalog if the file does not exist.
func (s *LocalSource) Load() (*Catalog, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Catalog{Version: 1, Modules: []Module{}}, nil
		}
		return nil, fmt.Errorf("failed to read catalog %s: %w", s.Path, err)
	}

	var cat Catalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("failed to parse catalog %s: %w", s.Path, err)
	}
	return &cat, nil
}

// Name returns the file path.
func (s *LocalSource) Name() string { return s.Path }

// ---------------------------------------------------------------------------
// HTTPSource — fetches catalog.yaml from an HTTP(S) URL.
// ---------------------------------------------------------------------------

// HTTPSource fetches a catalog from a remote HTTP(S) URL.
type HTTPSource struct {
	URL string
}

// Load fetches the catalog over HTTP.
func (s *HTTPSource) Load() (*Catalog, error) {
	resp, err := http.Get(s.URL) //nolint:gosec // user-provided URL is intentional
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote catalog %s: %w", s.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote catalog %s returned HTTP %d", s.URL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote catalog body: %w", err)
	}

	var cat Catalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("failed to parse remote catalog: %w", err)
	}
	return &cat, nil
}

// Name returns the URL.
func (s *HTTPSource) Name() string { return s.URL }

// ---------------------------------------------------------------------------
// SourceFor — factory helper
// ---------------------------------------------------------------------------

// isRemoteURL returns true if path looks like an HTTP(S) URL.
func isRemoteURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// SourceFor returns a CatalogSource appropriate for the given path.
// If the path starts with http:// or https://, an HTTPSource is returned.
// Otherwise a LocalSource is returned.
func SourceFor(path string) CatalogSource {
	if isRemoteURL(path) {
		return &HTTPSource{URL: path}
	}
	return &LocalSource{Path: path}
}
