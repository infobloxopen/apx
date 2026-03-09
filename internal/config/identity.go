package config

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

// ParseAPIID parses an API ID string into its constituent parts.
// Accepts two forms:
//   - 4-part: format/domain/name/line  (e.g. "proto/payments/ledger/v1")
//   - 3-part: format/name/line         (e.g. "proto/orders/v1" — no explicit domain)
func ParseAPIID(apiID string) (*APIIdentity, error) {
	parts := strings.Split(apiID, "/")
	if len(parts) < 3 || len(parts) > 4 {
		return nil, fmt.Errorf("invalid API ID %q: expected format/<name>/<line> or format/<domain>/<name>/<line>", apiID)
	}

	validFormats := map[string]bool{
		"proto": true, "openapi": true, "avro": true,
		"jsonschema": true, "parquet": true,
	}

	format := parts[0]
	if !validFormats[format] {
		return nil, fmt.Errorf("invalid API format %q: must be one of proto, openapi, avro, jsonschema, parquet", format)
	}

	var domain, name, line string
	if len(parts) == 4 {
		domain, name, line = parts[1], parts[2], parts[3]
	} else {
		// 3-part form: no explicit domain
		name, line = parts[1], parts[2]
	}

	if !isValidLine(line) {
		return nil, fmt.Errorf("invalid API line %q: must be v<major> (e.g. v1, v2)", line)
	}

	return &APIIdentity{
		ID:     apiID,
		Format: format,
		Domain: domain,
		Name:   name,
		Line:   line,
	}, nil
}

// isValidLine checks that a line string matches v<N> where N >= 0.
// v0 lines are allowed for experimental/beta APIs.
func isValidLine(line string) bool {
	if !strings.HasPrefix(line, "v") {
		return false
	}
	n, err := strconv.Atoi(line[1:])
	if err != nil || n < 0 {
		return false
	}
	return true
}

// LineMajor returns the major version number from a line string (e.g. "v1" → 1, "v0" → 0).
func LineMajor(line string) (int, error) {
	if !strings.HasPrefix(line, "v") {
		return 0, fmt.Errorf("line %q must start with 'v'", line)
	}
	n, err := strconv.Atoi(line[1:])
	if err != nil {
		return 0, fmt.Errorf("line %q is not a valid version: %w", line, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("line major version must be >= 0, got %d", n)
	}
	return n, nil
}

// IsV0Line returns true if the line represents a v0 (unstable) API line.
func IsV0Line(line string) bool {
	n, err := LineMajor(line)
	return err == nil && n == 0
}

// DeriveSourcePath computes the canonical source path for an API ID.
// For example: "proto/payments/ledger/v1" → "proto/payments/ledger/v1"
func DeriveSourcePath(apiID string) string {
	return apiID
}

// EffectiveSourcePath returns the filesystem path for an API.
// For first-party APIs, this equals the API ID (via DeriveSourcePath).
// For external APIs with a managed_path, the managed_path is used instead,
// since their filesystem layout differs from the canonical ID.
func EffectiveSourcePath(apiID, managedPath string) string {
	if managedPath != "" {
		return managedPath
	}
	return DeriveSourcePath(apiID)
}

// EffectiveGoRoot returns the import root to use for Go module/import path
// derivation. If a custom importRoot is configured, it takes precedence
// over the sourceRepo (Git hosting path). This supports organizations that
// use a vanity import root (e.g. go.acme.dev/apis) while hosting code at
// a different Git path (e.g. github.com/acme/apis).
func EffectiveGoRoot(sourceRepo, importRoot string) string {
	if importRoot != "" {
		return importRoot
	}
	return sourceRepo
}

// DeriveTag computes the git tag for a release of an API.
//
// Format: <api-id>/v<semver>
// Example: "proto/payments/ledger/v1" + "1.0.0-beta.1"
//
//	→ "proto/payments/ledger/v1/v1.0.0-beta.1"
func DeriveTag(apiID, version string) string {
	v := version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return apiID + "/" + v
}

// BuildIdentityBlock creates a full identity section from an API ID string
// and source repository. This is the primary entry point for populating
// identity fields from minimal inputs.
//
// Language coordinates are NOT derived here — use language.DeriveAllCoords
// separately to get per-language coordinates.
func BuildIdentityBlock(apiID, sourceRepo, lifecycle, currentVersion string) (*APIIdentity, *SourceIdentity, *ReleaseInfo, error) {
	api, err := ParseAPIID(apiID)
	if err != nil {
		return nil, nil, nil, err
	}
	if lifecycle != "" {
		api.Lifecycle = lifecycle
	}

	source := &SourceIdentity{
		Repo: sourceRepo,
		Path: DeriveSourcePath(apiID),
	}

	var release *ReleaseInfo
	if currentVersion != "" {
		release = &ReleaseInfo{Current: currentVersion}
	}

	return api, source, release, nil
}

// ValidateLifecycle checks if a lifecycle string is valid.
func ValidateLifecycle(lifecycle string) error {
	valid := map[string]bool{
		"experimental": true,
		"preview":      true,
		"beta":         true, // canonical; preview is the backward-compat alias
		"stable":       true,
		"deprecated":   true,
		"sunset":       true,
	}
	if !valid[lifecycle] {
		return fmt.Errorf("invalid lifecycle %q: must be one of experimental, beta, stable, deprecated, sunset", lifecycle)
	}
	return nil
}

// ValidateGoPackage checks that a go_package value matches the derived import path.
// The goPackage may include a ";alias" suffix which is stripped before comparison.
// Returns nil if the paths match, or an error describing the mismatch.
func ValidateGoPackage(goPackage string, expectedImport string) error {
	if goPackage == "" {
		return nil // No go_package set — skip validation
	}

	// Strip alias suffix (e.g. "path;alias" → "path")
	importPath := goPackage
	if idx := strings.Index(importPath, ";"); idx >= 0 {
		importPath = importPath[:idx]
	}

	if importPath != expectedImport {
		return fmt.Errorf("go_package mismatch: got %q, expected %q", importPath, expectedImport)
	}
	return nil
}

// DeriveGoModDir computes the directory (relative to repo root) where go.mod
// should be placed for the given API identity.
//
// Rules:
//   - For v0: <format>/<domain>/<name>        (module root above the /v0/ package dir)
//   - For v1: <format>/<domain>/<name>        (module root above the /v1/ package dir)
//   - For v2+: <format>/<domain>/<name>/v<N>   (module root = package dir, includes major version suffix)
func DeriveGoModDir(api *APIIdentity) string {
	base := path.Join(api.Format, api.Domain, api.Name)
	major, err := LineMajor(api.Line)
	if err != nil || major <= 1 {
		return base
	}
	return fmt.Sprintf("%s/v%d", base, major)
}

// ParseLineFromID extracts the line component from an API ID string.
// Handles both 4-part (format/domain/name/line) and 3-part (format/name/line) forms.
// For example: "proto/payments/ledger/v1" → "v1", "proto/orders/v2" → "v2"
// Returns "v1" as a safe default if parsing fails.
func ParseLineFromID(apiID string) string {
	parts := strings.Split(apiID, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	if len(parts) == 3 {
		return parts[2]
	}
	return "v1"
}
