package config

import (
	"fmt"
	"strings"
)

// Origin constant for sourced APIs (first-party, non-canonical path).
const OriginSourced = "sourced"

// APISource declares a remote repository whose git tags are scanned for
// API release tags during catalog generation. This enables first-party APIs
// that live in separate repositories with non-canonical directory layouts
// to appear in the catalog.
type APISource struct {
	Repo       string            `yaml:"repo"`                  // e.g. "github.com/Infoblox-CTO/ngp.authz"
	ImportMode string            `yaml:"import_mode,omitempty"` // "preserve" (default) or "rewrite"
	PathMap    map[string]string `yaml:"path_map,omitempty"`    // canonical API ID → actual source path
}

// Validate checks that the API source has valid fields.
func (s *APISource) Validate() error {
	if s.Repo == "" {
		return fmt.Errorf("repo is required")
	}
	if err := validateRepoURL(s.Repo); err != nil {
		return fmt.Errorf("invalid repo URL: %w", err)
	}

	// Default import_mode to preserve
	if s.ImportMode == "" {
		s.ImportMode = ImportModePreserve
	}
	if s.ImportMode != ImportModePreserve && s.ImportMode != ImportModeRewrite {
		return fmt.Errorf("%w: must be %q or %q", ErrExternalInvalidMode, ImportModePreserve, ImportModeRewrite)
	}

	// Validate path_map entries
	for apiID, path := range s.PathMap {
		if _, err := ParseAPIID(apiID); err != nil {
			return fmt.Errorf("invalid API ID in path_map key %q: %w", apiID, err)
		}
		if strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
			return fmt.Errorf("path_map value %q must not have leading or trailing slashes", path)
		}
		if strings.Contains(path, "..") {
			return fmt.Errorf("path_map value %q must not contain '..'", path)
		}
	}

	return nil
}

// SourcePathFor returns the effective source path for an API ID.
// If the path_map has an entry, it returns that; otherwise returns the API ID itself.
func (s *APISource) SourcePathFor(apiID string) string {
	if p, ok := s.PathMap[apiID]; ok {
		return p
	}
	return apiID
}
