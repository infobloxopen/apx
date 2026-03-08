package catalog

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"golang.org/x/mod/semver"
)

// MergeExternalAPIs converts external API registrations into Module entries
// and merges them into the catalog. It detects ID and path conflicts with
// first-party modules and replaces any existing external entries that match.
func MergeExternalAPIs(cat *Catalog, externals []config.ExternalRegistration) error {
	if len(externals) == 0 {
		return nil
	}

	// Build index of first-party module IDs and paths
	firstPartyIDs := make(map[string]bool)
	firstPartyPaths := make(map[string]string) // path → ID
	for _, m := range cat.Modules {
		if m.Origin == "" { // first-party
			firstPartyIDs[m.ID] = true
			firstPartyPaths[m.Path] = m.ID
		}
	}

	// Remove existing external entries (will be replaced)
	var kept []Module
	for _, m := range cat.Modules {
		if m.Origin == "" { // first-party — always keep
			kept = append(kept, m)
		}
		// External entries are dropped; they'll be re-added from externals list
	}

	// Convert and validate each external registration
	for _, ext := range externals {
		// Check for conflicts with first-party modules
		if firstPartyIDs[ext.ID] {
			return fmt.Errorf("external API %q conflicts with first-party module", ext.ID)
		}
		if ownerID, exists := firstPartyPaths[ext.ManagedPath]; exists {
			return fmt.Errorf("external API %q managed_path %q conflicts with first-party module %q",
				ext.ID, ext.ManagedPath, ownerID)
		}

		// Parse identity from the API ID
		identity, err := config.ParseAPIID(ext.ID)
		if err != nil {
			return fmt.Errorf("invalid external API ID %q: %w", ext.ID, err)
		}

		// Determine effective path
		effectivePath := config.EffectiveSourcePath(ext.ID, ext.ManagedPath)

		// Build Module from registration
		mod := Module{
			ID:           ext.ID,
			Name:         ext.ID,
			Format:       identity.Format,
			Domain:       identity.Domain,
			APILine:      identity.Line,
			Description:  ext.Description,
			Version:      ext.Version,
			Lifecycle:    ext.Lifecycle,
			Path:         effectivePath,
			Tags:         ext.Tags,
			Owners:       ext.Owners,
			Origin:       ext.Origin,
			ManagedRepo:  ext.ManagedRepo,
			UpstreamRepo: ext.UpstreamRepo,
			UpstreamPath: ext.UpstreamPath,
			ImportMode:   ext.ImportMode,
		}

		// Set version tracking fields
		if ext.Version != "" {
			if isPrerelease(ext.Version) {
				mod.LatestPrerelease = ext.Version
			} else {
				mod.LatestStable = ext.Version
			}
		}

		kept = append(kept, mod)
	}

	cat.Modules = kept
	return nil
}

// isPrerelease returns true if the version string is a semver prerelease.
func isPrerelease(version string) bool {
	v := version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return false
	}
	return semver.Prerelease(v) != ""
}

// ExternalModuleCount counts external modules in a catalog.
func ExternalModuleCount(cat *Catalog) (firstParty, external int) {
	for _, m := range cat.Modules {
		if m.Origin != "" {
			external++
		} else {
			firstParty++
		}
	}
	return
}
