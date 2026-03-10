// Package site generates a static API catalog explorer site from a catalog.
//
// The site is a single-page application with embedded HTML/CSS/JS that
// displays all APIs in the catalog with filtering, searching, and
// language-specific coordinate derivation.
//
// Usage:
//
//	apx catalog site generate --catalog=catalog/catalog.yaml --output=_site
package site

import (
	"path/filepath"
	"time"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/language"
	"github.com/infobloxopen/apx/internal/site/schema"
)

// SiteData is the top-level structure serialized to index.json.
type SiteData struct {
	Org         string     `json:"org"`
	Repo        string     `json:"repo"`
	ImportRoot  string     `json:"import_root,omitempty"`
	GeneratedAt string     `json:"generated_at"`
	APIs        []APIEntry `json:"apis"`
}

// APIEntry is the JSON-serialized form of one API in the catalog explorer.
type APIEntry struct {
	ID               string                     `json:"id"`
	Format           string                     `json:"format"`
	Domain           string                     `json:"domain,omitempty"`
	Name             string                     `json:"name,omitempty"`
	Line             string                     `json:"line,omitempty"`
	Description      string                     `json:"description,omitempty"`
	Version          string                     `json:"version,omitempty"`
	LatestStable     string                     `json:"latest_stable,omitempty"`
	LatestPrerelease string                     `json:"latest_prerelease,omitempty"`
	Lifecycle        string                     `json:"lifecycle,omitempty"`
	Compatibility    *CompatibilityInfo         `json:"compatibility,omitempty"`
	Tags             []string                   `json:"tags,omitempty"`
	Owners           []string                   `json:"owners,omitempty"`
	Origin           string                     `json:"origin,omitempty"`
	Languages        map[string][]LanguageCoord `json:"languages,omitempty"`
	Schema           *schema.SchemaDetail       `json:"schema,omitempty"`
}

// CompatibilityInfo describes the backward-compatibility contract.
type CompatibilityInfo struct {
	Level          string `json:"level"`
	Summary        string `json:"summary"`
	BreakingPolicy string `json:"breaking_policy"`
	ProductionUse  string `json:"production_use"`
}

// LanguageCoord is one line in a language's coordinate derivation.
type LanguageCoord struct {
	Label string `json:"label"` // e.g. "Go module", "Go import", "Py dist"
	Value string `json:"value"` // the coordinate value
}

// BuildSiteData converts a catalog into the site data structure,
// deriving language coordinates for every module.
// If repoDir is non-empty, schema files are extracted from the filesystem
// at each module's path relative to repoDir.
func BuildSiteData(cat *catalog.Catalog, sourceRepo, importRoot, org, repoDir string) *SiteData {
	data := &SiteData{
		Org:         cat.Org,
		Repo:        cat.Repo,
		ImportRoot:  cat.ImportRoot,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		APIs:        make([]APIEntry, 0, len(cat.Modules)),
	}

	for _, m := range cat.Modules {
		entry := buildAPIEntry(m, sourceRepo, importRoot, org)
		if entry != nil {
			// Extract schema content when repoDir is provided.
			if repoDir != "" && m.Path != "" {
				modulePath := filepath.Join(repoDir, m.Path)
				entry.Schema = schema.ExtractSchema(modulePath, m.Format)
			}
			data.APIs = append(data.APIs, *entry)
		}
	}

	return data
}

// buildAPIEntry converts a single catalog module into an APIEntry.
// Returns nil if the module ID cannot be parsed.
func buildAPIEntry(m catalog.Module, sourceRepo, importRoot, org string) *APIEntry {
	api, err := config.ParseAPIID(m.ID)
	if err != nil {
		// Skip modules with unparseable IDs (legacy or malformed entries).
		return nil
	}

	entry := &APIEntry{
		ID:               m.ID,
		Format:           m.Format,
		Domain:           m.Domain,
		Name:             api.Name,
		Line:             m.APILine,
		Description:      m.Description,
		Version:          m.Version,
		LatestStable:     m.LatestStable,
		LatestPrerelease: m.LatestPrerelease,
		Lifecycle:        config.NormalizeLifecycle(m.Lifecycle),
		Tags:             m.Tags,
		Owners:           m.Owners,
		Origin:           m.Origin,
	}

	// Enrich lifecycle compatibility info.
	if m.Lifecycle != "" {
		promise := config.DeriveCompatibilityPromise(m.APILine, m.Lifecycle)
		entry.Compatibility = &CompatibilityInfo{
			Level:          promise.Level,
			Summary:        promise.Summary,
			BreakingPolicy: promise.BreakingPolicy,
			ProductionUse:  config.ProductionRecommendation(m.Lifecycle),
		}
	}

	// Derive language coordinates — same pattern as show.go.
	if api.Lifecycle == "" && m.Lifecycle != "" {
		api.Lifecycle = m.Lifecycle
	}
	ctx := language.DerivationContext{
		SourceRepo: sourceRepo,
		ImportRoot: importRoot,
		Org:        org,
		API:        api,
	}
	coords, err := language.DeriveAllCoords(ctx)
	if err == nil && len(coords) > 0 {
		entry.Languages = make(map[string][]LanguageCoord)
		for _, p := range language.All() {
			c, ok := coords[p.Name()]
			if !ok {
				continue
			}
			var lc []LanguageCoord
			for _, rl := range p.ReportLines(c) {
				lc = append(lc, LanguageCoord{
					Label: rl.Label,
					Value: rl.Value,
				})
			}
			entry.Languages[p.Name()] = lc
		}
	}

	return entry
}
