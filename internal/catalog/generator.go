package catalog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/infobloxopen/apx/internal/validator"
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
	// ResourceTypes lists the AIP-122 resource types this module declares
	// (read from google.api.resource annotations in its protos at catalog
	// generation). It is the key that type→module resolution reads.
	ResourceTypes []string `yaml:"resource_types,omitempty"`
	// CRD facts (populated for the crd format at catalog generation). They make
	// a Kubernetes GVK a first-class, version-constrainable capability token.
	CRDGroup       string   `yaml:"crd_group,omitempty"`
	CRDKind        string   `yaml:"crd_kind,omitempty"`
	CRDScope       string   `yaml:"crd_scope,omitempty"`
	ServedVersions []string `yaml:"served_versions,omitempty"`
	StorageVersion string   `yaml:"storage_version,omitempty"`
	// External API provenance fields (empty for first-party APIs)
	Origin       string `yaml:"origin,omitempty"`        // "external" or "forked"
	ManagedRepo  string `yaml:"managed_repo,omitempty"`  // internal curating repository
	UpstreamRepo string `yaml:"upstream_repo,omitempty"` // original external repository
	UpstreamPath string `yaml:"upstream_path,omitempty"` // path in upstream repository
	ImportMode   string `yaml:"import_mode,omitempty"`   // "preserve" or "rewrite"
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
	Version     int      `yaml:"version"`
	Org         string   `yaml:"org"`
	Repo        string   `yaml:"repo"`
	ImportRoot  string   `yaml:"import_root,omitempty"`
	GeneratedBy string   `yaml:"generated_by,omitempty"`
	Modules     []Module `yaml:"modules"`
}

// Generator handles catalog generation
type Generator struct {
	catalogPath string

	// Source, if non-nil, is used by Load() instead of catalogPath.
	// This allows callers to wire in registry, cached, or aggregate sources.
	Source CatalogSource
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

// Load loads the existing catalog. If Source is set, it delegates to that.
// Otherwise it uses SourceFor(catalogPath) which routes to LocalSource or
// HTTPSource based on the path prefix.
func (g *Generator) Load() (*Catalog, error) {
	if g.Source != nil {
		return g.Source.Load()
	}
	return SourceFor(g.catalogPath).Load()
}

// LoadFrom loads the catalog from an explicit CatalogSource, ignoring
// the generator's catalogPath. Useful when the caller has already
// resolved which source to use (e.g. registry, cache, aggregate).
func (g *Generator) LoadFrom(src CatalogSource) (*Catalog, error) {
	return src.Load()
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
		// A CRD is recognizable by content, so sniff before dir heuristics.
		if ext == ".yaml" || ext == ".yml" {
			if data, err := os.ReadFile(path); err == nil && validator.LooksLikeCRD(data) {
				return "crd"
			}
		}
		if strings.Contains(dir, "openapi") {
			return "openapi"
		}
		if strings.Contains(dir, "jsonschema") {
			return "jsonschema"
		}
		if strings.Contains(dir, "avro") {
			return "avro"
		}
		if strings.Contains(dir, "crd") {
			return "crd"
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
			// Enrich CRD modules with their GVK and served/storage facts so a
			// Kubernetes capability is version-constrainable from the catalog.
			if format == "crd" {
				if info, infoErr := validator.ExtractCRDInfo(path); infoErr == nil {
					module.CRDGroup = info.Group
					module.CRDKind = info.Kind
					module.CRDScope = info.Scope
					module.ServedVersions = info.ServedVersions
					module.StorageVersion = info.StorageVersion
				}
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
		"jsonschema": true, "parquet": true, "crd": true,
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

// k8sVersionLineRe matches a Kubernetes API version segment used by the crd
// format: v<major> with an optional alpha/beta maturity suffix (e.g. v1alpha1).
var k8sVersionLineRe = regexp.MustCompile(`^v[1-9][0-9]*(?:(?:alpha|beta)[1-9][0-9]*)?$`)

// isVersionLine checks if a string is a valid API line segment. It accepts the
// canonical "v<N>" form (N >= 1, no leading zeros) and the Kubernetes
// "v<major>[alpha|beta<n>]" form used by the crd format.
func isVersionLine(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	digits := s[1:]
	// Reject leading zeros (e.g. "v01")
	if len(digits) > 1 && digits[0] == '0' {
		return false
	}
	allDigits := true
	n := 0
	for _, c := range digits {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
		n = n*10 + int(c-'0')
	}
	if allDigits {
		return n >= 1
	}
	// Kubernetes version with alpha/beta suffix.
	return k8sVersionLineRe.MatchString(s)
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

// ParseReleaseTag parses a release git tag into an API ID and version string.
// Returns ("", "") for non-matching tags. Uses golang.org/x/mod/semver for
// robust semver validation including pre-releases, build metadata, and proper
// precedence ordering per the semver 2.0.0 spec.
//
// It accepts the two tag shapes `apx release finalize` actually mints, which
// differ because Go-module tag semantics drop the /v0 and /v1 major-version
// suffix from a module path:
//
//   - line-present (v2+):  <format>/<domain>/<name>/<line>/v<semver>   (5 segments)
//     e.g. "openapi/csp.infoblox.com/probe/v2/v2.0.0"
//   - line-dropped (v0/v1): <format>/<domain>/<name>/v<semver>          (4 segments)
//     e.g. "openapi/csp.infoblox.com/probe/v1.0.0"
//
// For the line-dropped form the API line is recovered from the version's major
// (valid because the suffix is only dropped for major <= 1, where the release's
// semver major equals its line). Before this accepted both shapes, every v0/v1
// module minted by finalize was silently absent from the generated catalog.
func ParseReleaseTag(tag string) (apiID string, version string) {
	parts := strings.Split(tag, "/")

	validFormats := map[string]bool{
		"proto": true, "openapi": true, "avro": true,
		"jsonschema": true, "parquet": true, "crd": true,
	}

	switch len(parts) {
	case 5:
		// Line-present form: <format>/<domain>/<name>/<line>/v<semver>.
		if !validFormats[parts[0]] {
			return "", ""
		}
		if !isVersionLine(parts[3]) {
			return "", ""
		}
		if !semver.IsValid(parts[4]) {
			return "", ""
		}
		apiID = fmt.Sprintf("%s/%s/%s/%s", parts[0], parts[1], parts[2], parts[3])
		return apiID, parts[4]

	case 4:
		// Line-dropped form: <format>/<domain>/<name>/v<semver>.
		if !validFormats[parts[0]] {
			return "", ""
		}
		// Reject the domainless line-present shape (<format>/<name>/<line>/v<semver>):
		// catalog IDs are 4-part, domain-qualified (see detectAPIIdentity), so a
		// version-line in the name position is not a supported catalog ID.
		if isVersionLine(parts[2]) {
			return "", ""
		}
		v := parts[3]
		if !semver.IsValid(v) {
			return "", ""
		}
		line := semver.Major(v) // "v0" or "v1" for a line-dropped tag
		if line == "" {
			return "", ""
		}
		apiID = fmt.Sprintf("%s/%s/%s/%s", parts[0], parts[1], parts[2], line)
		return apiID, v

	default:
		return "", ""
	}
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
	LatestStable     string            // empty if none
	LatestPrerelease string            // empty if none
	lifecycleByVer   map[string]string // recorded lifecycle per version (from the tag annotation)
	tags             []string          // union of recorded tags across all versions
}

// GenerateFromTags builds a Catalog from a list of git tag names, deriving
// lifecycle from semver. It is retained for callers (and tests) that only have
// tag names; when annotation metadata is available prefer GenerateFromTagRecords
// so a recorded lifecycle (e.g. deprecated) and first-party tags survive.
func GenerateFromTags(tags []string, org, repo string) *Catalog {
	records := make([]TagRecord, len(tags))
	for i, t := range tags {
		records[i] = TagRecord{Tag: t}
	}
	return GenerateFromTagRecords(records, org, repo)
}

// GenerateFromTagRecords builds a Catalog from release tags plus the metadata
// recorded in each tag's annotation. Tags that don't match the release pattern
// are silently skipped. Version comparison uses golang.org/x/mod/semver, which
// fully implements semver 2.0.0 precedence including pre-release ordering.
//
// A module's lifecycle is the lifecycle recorded on its CURRENT version (latest
// stable, else latest prerelease) when the annotation carries one; otherwise it
// falls back to the semver-derived heuristic. This is what makes a module cut
// with `--lifecycle deprecated` show as deprecated rather than stable (F-32).
// Recorded first-party tags are unioned across the module's versions (F-33).
func GenerateFromTagRecords(records []TagRecord, org, repo string) *Catalog {
	apis := make(map[string]*apiAccum)

	for _, rec := range records {
		apiID, version := ParseReleaseTag(rec.Tag)
		if apiID == "" {
			continue
		}

		acc, ok := apis[apiID]
		if !ok {
			parts := strings.Split(apiID, "/")
			acc = &apiAccum{
				Format:         parts[0],
				Domain:         parts[1],
				APILine:        parts[3],
				Path:           apiID,
				lifecycleByVer: map[string]string{},
			}
			apis[apiID] = acc
		}

		if rec.Lifecycle != "" {
			acc.lifecycleByVer[version] = rec.Lifecycle
		}
		acc.tags = UnionTags(acc.tags, rec.Tags)

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
			Tags:    acc.tags,
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
			m.Lifecycle = lifecycleFromPrerelease(acc.LatestPrerelease)
		}
		// A lifecycle recorded on the current version's release tag overrides the
		// semver-derived value — this is how `--lifecycle deprecated` (and any
		// non-derivable state) surfaces in the generated catalog (F-32).
		if rec := acc.lifecycleByVer[m.Version]; rec != "" {
			m.Lifecycle = rec
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

// lifecycleFromPrerelease maps a prerelease suffix to a lifecycle state.
func lifecycleFromPrerelease(version string) string {
	pre := strings.ToLower(semver.Prerelease(version)) // includes leading "-", e.g. "-beta.1"
	switch {
	case strings.HasPrefix(pre, "-alpha"):
		return "experimental"
	case strings.HasPrefix(pre, "-beta"):
		return "beta"
	case strings.HasPrefix(pre, "-rc"):
		return "beta"
	default:
		return "experimental"
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
