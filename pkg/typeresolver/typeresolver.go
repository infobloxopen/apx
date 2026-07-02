// Package typeresolver is the public, importable entry point for resolving an
// AIP-122 resource type to the catalog module that serves it.
//
// It is the catalog-backed implementation of the devedge-sdk F041
// ReferenceResolver seam (WS-021 P1): a consumer supplies a catalog source
// (local file, published OCI registry, HTTP, or an aggregate of these) and a
// resource type string taken from a google.api.resource_reference.type, and
// receives the serving module's path coordinates. Resolution fails loud on an
// unknown or ambiguous type; match ErrUnresolved / ErrAmbiguous with errors.Is.
//
// The surface is intentionally minimal: one function, one result type, two
// sentinel errors.
package typeresolver

import (
	"github.com/infobloxopen/apx/internal/catalog"
)

// Resolution is the outcome of a successful type resolution: path coordinates
// for the serving module. The concrete network host is not part of it — apx is
// a schema catalog, not a service registry.
type Resolution = catalog.Resolution

// Source is a catalog origin (local file, OCI registry, HTTP, or aggregate).
// Construct one with catalog.SourceFor / catalog.RegistrySource, etc.
type Source = catalog.CatalogSource

// Sentinel errors, re-exported so consumers can match without importing the
// internal package.
var (
	// ErrUnresolved means no module in the catalog serves the requested type.
	ErrUnresolved = catalog.ErrUnresolved
	// ErrAmbiguous means more than one module serves the requested type.
	ErrAmbiguous = catalog.ErrAmbiguous
)

// Resolve loads the catalog from src and resolves resourceType to its serving
// module. It returns ErrUnresolved for an unknown type and ErrAmbiguous when
// more than one module claims the type. A declared-but-unserved type resolves
// successfully with a non-empty Resolution.Warning.
func Resolve(src Source, resourceType string) (*Resolution, error) {
	cat, err := src.Load()
	if err != nil {
		return nil, err
	}
	return catalog.ResolveType(cat, resourceType)
}
