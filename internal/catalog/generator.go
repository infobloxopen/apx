package catalog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// Module represents a schema module in the catalog
type Module struct {
	ID               string   `yaml:"id"`             // canonical API ID, e.g. "proto/payments/ledger/v1"
	Name             string   `yaml:"name,omitempty"` // backward compat (deprecated, use ID)
	Format           string   `yaml:"format"`
	Domain           string   `yaml:"domain,omitempty"`   // e.g. "payments"
	APILine          string   `yaml:"api_line,omitempty"` // e.g. "v1"
	Description      string   `yaml:"description,omitempty"`
	Version          string   `yaml:"version,omitempty"`
	LatestStable     string   `yaml:"latest_stable,omitempty"`
	LatestPrerelease string   `yaml:"latest_prerelease,omitempty"`
	Lifecycle        string   `yaml:"lifecycle,omitempty"`      // experimental, beta, stable, deprecated, sunset
	Compatibility    string   `yaml:"compatibility,omitempty"`  // none, stabilizing, full, maintenance, eol
	ProductionUse    string   `yaml:"production_use,omitempty"` // human-readable recommendation
	Path             string   `yaml:"path"`
	Tags             []string `yaml:"tags,omitempty"`
	Owners           []string `yaml:"owners,omitempty"`
}

// DisplayName returns the best identifier for display: ID if set, otherwise Name.
func (m Module) DisplayName() string {
	if m.ID != "" {
		return m.ID
	}
	return m.Name
}

// Catalog represents the schema catalog
type Catalog struct {
	Version int      `yaml:"version"`
	Org     string   `yaml:"org"`
	Repo    string   `yaml:"repo"`
	Modules []Module `yaml:"modules"`
}

// Generator handles catalog generation
type Generator struct {
	catalogPath string
}

// NewGenerator creates a new catalog generator
func NewGenerator(catalogPath string) *Generator {
	if catalogPath == "" {
		catalogPath = "catalog.yaml"
	}
	return &Generator{
		catalogPath: catalogPath,
	}
}

// Load loads the existing catalog
func (g *Generator) Load() (*Catalog, error) {
	data, err := os.ReadFile(g.catalogPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty catalog
			return &Catalog{
				Version: 1,
				Modules: []Module{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read catalog: %w", err)
	}

	var catalog Catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}

	return &catalog, nil
}

// Save saves the catalog to disk
func (g *Generator) Save(catalog *Catalog) error {
	data, err := yaml.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	if err := os.WriteFile(g.catalogPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write catalog: %w", err)
	}

	return nil
}

// AddModule adds a module to the catalog
func (g *Generator) AddModule(module Module) error {
	catalog, err := g.Load()
	if err != nil {
		return err
	}

	// Check if module already exists (match on ID first, fall back to Name)
	for i, m := range catalog.Modules {
		if (module.ID != "" && m.ID == module.ID) || (module.ID == "" && m.Name == module.Name) {
			// Update existing module
			catalog.Modules[i] = module
			return g.Save(catalog)
		}
	}

	// Add new module
	catalog.Modules = append(catalog.Modules, module)
	return g.Save(catalog)
}

// RemoveModule removes a module from the catalog by ID or Name.
func (g *Generator) RemoveModule(name string) error {
	catalog, err := g.Load()
	if err != nil {
		return err
	}

	for i, m := range catalog.Modules {
		if m.ID == name || m.Name == name {
			catalog.Modules = append(catalog.Modules[:i], catalog.Modules[i+1:]...)
			return g.Save(catalog)
		}
	}

	return fmt.Errorf("module not found: %s", name)
}

// DetectFormat detects the schema format from file path
func DetectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	dir := filepath.Dir(path)

	switch ext {
	case ".proto":
		return "proto"
	case ".avsc", ".avdl", ".avpr":
		return "avro"
	case ".parquet":
		return "parquet"
	case ".yaml", ".yml", ".json":
		if strings.Contains(dir, "openapi") {
			return "openapi"
		}
		if strings.Contains(dir, "jsonschema") {
			return "jsonschema"
		}
		if strings.Contains(dir, "avro") {
			return "avro"
		}
		return "unknown"
	}

	return "unknown"
}

// ScanDirectory scans a directory for schema modules.
// It detects API identity from directory structure when the path matches
// the canonical pattern: <format>/<domain>/<name>/<line>/
func (g *Generator) ScanDirectory(dir string) ([]Module, error) {
	modules := []Module{}
	seen := make(map[string]bool) // track API IDs to avoid duplicates from multiple files

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		format := DetectFormat(path)
		if format == "unknown" {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Try to detect API identity from directory structure
		apiID, domain, apiLine := detectAPIIdentity(relPath)

		if apiID != "" {
			// Group by API ID — one module entry per API, not per file
			if seen[apiID] {
				return nil
			}
			seen[apiID] = true

			module := Module{
				ID:      apiID,
				Name:    apiID, // backward compat
				Format:  format,
				Domain:  domain,
				APILine: apiLine,
				Path:    filepath.Dir(relPath),
			}
			modules = append(modules, module)
		} else {
			// Legacy: no identity detected, use file-level entry
			module := Module{
				Name:   filepath.Base(path),
				Format: format,
				Path:   relPath,
			}
			modules = append(modules, module)
		}

		return nil
	})

	return modules, err
}

// detectAPIIdentity attempts to extract API identity from a relative path.
// It looks for the pattern: <format>/<domain>/<name>/<line>/...
// Returns (apiID, domain, line) or ("", "", "") if not matched.
func detectAPIIdentity(relPath string) (string, string, string) {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) < 4 {
		return "", "", ""
	}

	format := parts[0]
	validFormats := map[string]bool{
		"proto": true, "openapi": true, "avro": true,
		"jsonschema": true, "parquet": true,
	}
	if !validFormats[format] {
		return "", "", ""
	}

	// Check that the line segment matches v<N>
	line := parts[3]
	if !isVersionLine(line) {
		return "", "", ""
	}

	domain := parts[1]
	name := parts[2]
	apiID := fmt.Sprintf("%s/%s/%s/%s", format, domain, name, line)

	return apiID, domain, line
}

// isVersionLine checks if a string matches "v<N>" where N >= 1 with no leading zeros.
func isVersionLine(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	digits := s[1:]
	// Reject leading zeros (e.g. "v01")
	if len(digits) > 1 && digits[0] == '0' {
		return false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return false
		}
	}
	n := 0
	for _, c := range digits {
		n = n*10 + int(c-'0')
	}
	return n >= 1
}

// GenerateCatalog generates a catalog from a directory
func (g *Generator) GenerateCatalog(dir, org, repo string) error {
	modules, err := g.ScanDirectory(dir)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	catalog := &Catalog{
		Version: 1,
		Org:     org,
		Repo:    repo,
		Modules: modules,
	}

	return g.Save(catalog)
}

// Search searches the catalog for modules matching a query
func (g *Generator) Search(query string) ([]Module, error) {
	catalog, err := g.Load()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	matches := []Module{}

	for _, module := range catalog.Modules {
		if strings.Contains(strings.ToLower(module.DisplayName()), queryLower) ||
			strings.Contains(strings.ToLower(module.Description), queryLower) {
			matches = append(matches, module)
		}
	}

	return matches, nil
}

// ---------- Tag-based catalog generation ----------

// ParseReleaseTag parses a git tag matching `<format>/<domain>/<name>/<line>/v<semver>`
// into an API ID and the version string. Returns ("", "") for non-matching tags.
// Uses golang.org/x/mod/semver for robust semver validation including pre-releases,
// build metadata, and proper precedence ordering per the semver 2.0.0 spec.
func ParseReleaseTag(tag string) (apiID string, version string) {
	parts := strings.Split(tag, "/")
	if len(parts) != 5 {
		return "", ""
	}

	format := parts[0]
	validFormats := map[string]bool{
		"proto": true, "openapi": true, "avro": true,
		"jsonschema": true, "parquet": true,
	}
	if !validFormats[format] {
		return "", ""
	}

	line := parts[3]
	if !isVersionLine(line) {
		return "", ""
	}

	version = parts[4]
	if !semver.IsValid(version) {
		return "", ""
	}

	apiID = fmt.Sprintf("%s/%s/%s/%s", parts[0], parts[1], parts[2], parts[3])
	return apiID, version
}

// isStableVersion returns true if the version has no pre-release suffix.
func isStableVersion(v string) bool {
	return semver.Prerelease(v) == ""
}

// apiAccum accumulates version information for one API ID during catalog generation.
type apiAccum struct {
	Format           string
	Domain           string
	APILine          string
	Path             string
	LatestStable     string // empty if none
	LatestPrerelease string // empty if none
}

// GenerateFromTags builds a Catalog from a list of git tags.
// Tags that don't match the release pattern are silently skipped.
// Version comparison uses golang.org/x/mod/semver which fully implements
// semver 2.0.0 precedence rules including pre-release ordering.
func GenerateFromTags(tags []string, org, repo string) *Catalog {
	apis := make(map[string]*apiAccum)

	for _, tag := range tags {
		apiID, version := ParseReleaseTag(tag)
		if apiID == "" {
			continue
		}

		acc, ok := apis[apiID]
		if !ok {
			parts := strings.Split(apiID, "/")
			acc = &apiAccum{
				Format:  parts[0],
				Domain:  parts[1],
				APILine: parts[3],
				Path:    apiID,
			}
			apis[apiID] = acc
		}

		if isStableVersion(version) {
			if acc.LatestStable == "" || semver.Compare(version, acc.LatestStable) > 0 {
				acc.LatestStable = version
			}
		} else {
			if acc.LatestPrerelease == "" || semver.Compare(version, acc.LatestPrerelease) > 0 {
				acc.LatestPrerelease = version
			}
		}
	}

	// Build sorted module list
	ids := make([]string, 0, len(apis))
	for id := range apis {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	modules := make([]Module, 0, len(ids))
	for _, id := range ids {
		acc := apis[id]
		m := Module{
			ID:      id,
			Name:    id,
			Format:  acc.Format,
			Domain:  acc.Domain,
			APILine: acc.APILine,
			Path:    acc.Path,
		}
		if acc.LatestStable != "" {
			m.LatestStable = acc.LatestStable
			m.Version = acc.LatestStable
			m.Lifecycle = "stable"
		}
		if acc.LatestPrerelease != "" {
			m.LatestPrerelease = acc.LatestPrerelease
			// If no stable release yet, version = latest prerelease
			if m.Version == "" {
				m.Version = acc.LatestPrerelease
			}
		}
		// If no stable version, infer lifecycle from prerelease
		if m.Lifecycle == "" && m.LatestPrerelease != "" {
			pre := strings.ToLower(semver.Prerelease(acc.LatestPrerelease))
			// pre includes the leading "-", e.g. "-beta.1"
			switch {
			case strings.HasPrefix(pre, "-alpha"):
				m.Lifecycle = "experimental"
			case strings.HasPrefix(pre, "-beta"):
				m.Lifecycle = "beta"
			case strings.HasPrefix(pre, "-rc"):
				m.Lifecycle = "beta"
			default:
				m.Lifecycle = "experimental"
			}
		}
		modules = append(modules, m)
	}

	return &Catalog{
		Version: 1,
		Org:     org,
		Repo:    repo,
		Modules: modules,
	}
}

// ListGitTags runs `git tag -l` in the given directory and returns all tags.
func ListGitTags(repoDir string) ([]string, error) {
	cmd := exec.Command("git", "tag", "-l")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list git tags: %w", err)
	}

	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// GenerateCatalogFromTags scans git tags in repoDir and writes a catalog.
func (g *Generator) GenerateCatalogFromTags(repoDir, org, repo string) error {
	tags, err := ListGitTags(repoDir)
	if err != nil {
		return err
	}

	cat := GenerateFromTags(tags, org, repo)
	return g.Save(cat)
}
