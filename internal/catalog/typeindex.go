package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Sentinel errors for type resolution. Consumers (e.g. the devedge-sdk F041
// ReferenceResolver) match on these with errors.Is to distinguish the two
// fail-loud outcomes from an unexpected failure.
var (
	// ErrUnresolved means no module in the catalog declares the requested type.
	ErrUnresolved = errors.New("unresolved resource reference")
	// ErrAmbiguous means more than one module declares the requested type.
	ErrAmbiguous = errors.New("ambiguous resource type")
)

// Resolution is the result of resolving a resource type to its serving module.
// It carries path coordinates only — apx resolves type → module (+
// domain/api_line, from which the API path is derivable); the concrete network
// host stays consumer/environment-supplied.
type Resolution struct {
	Type        string `json:"type"`
	ModuleID    string `json:"module_id"`
	Domain      string `json:"domain,omitempty"`
	APILine     string `json:"api_line,omitempty"`
	Version     string `json:"version,omitempty"`
	Lifecycle   string `json:"lifecycle,omitempty"`
	Origin      string `json:"origin,omitempty"`
	ManagedRepo string `json:"managed_repo,omitempty"`
	// Warning is non-empty when resolution succeeded but the consumer should be
	// cautioned — currently only "no serving surface" for a declared-but-unserved
	// type (schema-only module). It is advisory, not an error.
	Warning string `json:"warning,omitempty"`
}

// noServingSurfaceWarning is surfaced when a type resolves to a module that
// declares the type but exposes no serving surface (schema-only module).
const noServingSurfaceWarning = "no serving surface: type is declared but no service serves it"

// BuildTypeIndex maps each declared resource type to the module(s) that claim
// it. Claims are deduplicated by module ID so the same module listed twice
// (e.g. merged from two sources) counts once.
func BuildTypeIndex(cat *Catalog) map[string][]Module {
	index := make(map[string][]Module)
	if cat == nil {
		return index
	}
	for _, m := range cat.Modules {
		for _, t := range m.ResourceTypes {
			if t == "" {
				continue
			}
			if claimedBy(index[t], m.ID) {
				continue
			}
			index[t] = append(index[t], m)
		}
	}
	return index
}

// claimedBy reports whether a module with the given ID already appears in mods.
func claimedBy(mods []Module, id string) bool {
	for _, m := range mods {
		if m.ID == id {
			return true
		}
	}
	return false
}

// ResolveType resolves an AIP-122 resource type to the single module that
// serves it, returning path coordinates. It fails loud:
//   - zero claimants → ErrUnresolved (naming the type)
//   - more than one distinct claimant → ErrAmbiguous (listing the claimants)
//
// External/forked-imported modules resolve to the managing module: the
// Resolution carries ManagedRepo so the consumer calls the curated surface.
// A declared-but-unserved type resolves successfully with a "no serving
// surface" Warning rather than an error.
func ResolveType(cat *Catalog, resourceType string) (*Resolution, error) {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		return nil, fmt.Errorf("%w: empty resource type", ErrUnresolved)
	}

	claimants := BuildTypeIndex(cat)[resourceType]

	switch len(claimants) {
	case 0:
		return nil, fmt.Errorf("%w: no module in the catalog serves type %q", ErrUnresolved, resourceType)
	case 1:
		return resolutionFor(resourceType, claimants[0]), nil
	default:
		ids := make([]string, 0, len(claimants))
		for _, m := range claimants {
			ids = append(ids, m.DisplayName())
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("%w: type %q is claimed by %d modules: %s",
			ErrAmbiguous, resourceType, len(claimants), strings.Join(ids, ", "))
	}
}

// resolutionFor builds a Resolution from the serving module, applying the
// external/forked → managing-module rule and the unserved-type warning.
func resolutionFor(resourceType string, m Module) *Resolution {
	r := &Resolution{
		Type:        resourceType,
		ModuleID:    m.DisplayName(),
		Domain:      m.Domain,
		APILine:     m.APILine,
		Version:     m.Version,
		Lifecycle:   m.Lifecycle,
		Origin:      m.Origin,
		ManagedRepo: m.ManagedRepo,
	}
	if !hasServingSurface(m) {
		r.Warning = noServingSurfaceWarning
	}
	return r
}

// hasServingSurface reports whether the module exposes a serving surface.
// A schema-only module (no released version) has none. External/forked and
// sourced modules always resolve through their managing repo, so they are
// treated as served.
func hasServingSurface(m Module) bool {
	if m.Origin != "" {
		return true
	}
	return m.Version != "" || m.LatestStable != "" || m.LatestPrerelease != ""
}
