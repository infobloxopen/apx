package config

import (
	"fmt"
	"path"
	"regexp"
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
		"jsonschema": true, "parquet": true, "crd": true,
	}

	format := parts[0]
	if !validFormats[format] {
		return nil, fmt.Errorf("invalid API format %q: must be one of proto, openapi, avro, jsonschema, parquet, crd", format)
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

// k8sLineRe matches a Kubernetes-style API version line: v<major> with an
// optional alpha/beta maturity suffix (e.g. v1alpha1, v2beta3). It is used by
// the crd format, whose API line is the CRD version name. Plain v<N> lines
// (all other formats) are handled directly by strconv.
var k8sLineRe = regexp.MustCompile(`^v([1-9][0-9]*)(?:(?:alpha|beta)[1-9][0-9]*)?$`)

// isValidLine checks that a line string is a valid API line. It accepts the
// canonical v<N> form (N >= 0, used by proto/openapi/avro/…) and the
// Kubernetes v<major>[alpha|beta<n>] form (used by the crd format).
func isValidLine(line string) bool {
	if !strings.HasPrefix(line, "v") {
		return false
	}
	if n, err := strconv.Atoi(line[1:]); err == nil && n >= 0 {
		return true
	}
	return k8sLineRe.MatchString(line)
}

// LineMajor returns the major version number from a line string
// (e.g. "v1" → 1, "v0" → 0, and the Kubernetes forms "v1alpha1" → 1,
// "v2beta3" → 2).
func LineMajor(line string) (int, error) {
	if !strings.HasPrefix(line, "v") {
		return 0, fmt.Errorf("line %q must start with 'v'", line)
	}
	if n, err := strconv.Atoi(line[1:]); err == nil {
		if n < 0 {
			return 0, fmt.Errorf("line major version must be >= 0, got %d", n)
		}
		return n, nil
	}
	if m := k8sLineRe.FindStringSubmatch(line); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n, nil
	}
	return 0, fmt.Errorf("line %q is not a valid version", line)
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

// DeriveTagPrefix computes the git-tag prefix for an API. Go's module tag
// convention OMITS the major-version subdirectory from the tag prefix for ALL
// majors: a /vN module (N>=2) is tagged "<format>/<domain>/<name>/vX.Y.Z", never
// "<format>/<domain>/<name>/vN/vX.Y.Z". Requiring the /vN in the tag makes the
// module un-`go get`-able (go list -m finds nothing). So the prefix is always
// "<format>/<domain>/<name>" — DeriveGoModDir with any trailing /vN dropped —
// even though the Go MODULE PATH (DeriveGoModDir) correctly keeps /vN for v2+.
//
// CRD modules are the exception: they are one-per-version and are not Go
// modules, so their tag prefix keeps the full Kubernetes version segment (e.g.
// crd/g/k/v1alpha1) to remain independently taggable and catalog-resolvable.
//
// Falls back to the raw api-id if it cannot be parsed.
func DeriveTagPrefix(apiID string) string {
	api, err := ParseAPIID(apiID)
	if err != nil {
		return apiID
	}
	if api.Format == "crd" {
		return DeriveGoModDir(api)
	}
	return path.Join(api.Format, api.Domain, api.Name)
}

// DeriveTag computes the git tag for a release of an API.
//
// Format: <tag-prefix>/v<semver>, where <tag-prefix> is the major-version-
// stripped module path (see DeriveTagPrefix). Examples:
//   - "proto/payments/ledger/v1" + "1.0.0-beta.1" → "proto/payments/ledger/v1.0.0-beta.1"
//   - "proto/payments/ledger/v2" + "2.0.0"        → "proto/payments/ledger/v2.0.0"
func DeriveTag(apiID, version string) string {
	v := version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return DeriveTagPrefix(apiID) + "/" + v
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
	// CRD modules are one-per-version and are not Go modules. Their tag prefix
	// keeps the full Kubernetes version segment (e.g. crd/g/k/v1alpha1) so each
	// version is an independently taggable, catalog-resolvable module.
	if api.Format == "crd" {
		return path.Join(base, api.Line)
	}
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
