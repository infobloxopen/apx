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

// DeriveGoModule computes the Go module path for the given API line.
//
// Rules (per Go module versioning):
//   - For v0: <sourceRepo>/<format>/<domain>/<name>       (no version suffix)
//   - For v1: <sourceRepo>/<format>/<domain>/<name>       (no version suffix)
//   - For v2+: <sourceRepo>/<format>/<domain>/<name>/v<N>  (major version suffix)
func DeriveGoModule(sourceRepo string, api *APIIdentity) (string, error) {
	major, err := LineMajor(api.Line)
	if err != nil {
		return "", err
	}

	base := path.Join(sourceRepo, api.Format, api.Domain, api.Name)
	if major <= 1 {
		return base, nil
	}
	return fmt.Sprintf("%s/v%d", base, major), nil
}

// DeriveGoImport computes the Go import path for the given API line.
//
// Rules:
//   - For v0: <sourceRepo>/<format>/<domain>/<name>/v0      (v0 in import path)
//   - For v1: <sourceRepo>/<format>/<domain>/<name>/v1      (v1 in import path)
//   - For v2+: <sourceRepo>/<format>/<domain>/<name>/v<N>    (same as module path)
func DeriveGoImport(sourceRepo string, api *APIIdentity) (string, error) {
	major, err := LineMajor(api.Line)
	if err != nil {
		return "", err
	}

	base := path.Join(sourceRepo, api.Format, api.Domain, api.Name)
	return fmt.Sprintf("%s/v%d", base, major), nil
}

// DerivePythonDistName computes a PEP 625 distribution name for a Python package.
//
// Rules:
//   - Combines org, domain (if present), API name, and line
//   - All lowercase, joined with hyphens
//   - Example: org="acme", proto/payments/ledger/v1 → "acme-payments-ledger-v1"
//   - Example: org="acme", proto/orders/v1 (3-part, no domain) → "acme-orders-v1"
func DerivePythonDistName(org string, api *APIIdentity) string {
	parts := []string{strings.ToLower(org)}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	return strings.Join(parts, "-")
}

// DerivePythonImport computes a dotted Python import path for an API.
//
// Rules:
//   - Top-level namespace: {org}_apis (underscore-joined, Python identifier safe)
//   - Sub-packages: domain (if present), name, line
//   - Example: org="acme", proto/payments/ledger/v1 → "acme_apis.payments.ledger.v1"
//   - Example: org="acme", proto/orders/v1 → "acme_apis.orders.v1"
func DerivePythonImport(org string, api *APIIdentity) string {
	namespace := strings.ToLower(org) + "_apis"
	parts := []string{namespace}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	return strings.Join(parts, ".")
}

// DeriveMavenGroupId computes the Maven groupId for an organization.
//
// Rules:
//   - Pattern: com.<org>.apis
//   - Lowercased, hyphens replaced with dots
//   - Example: org="acme" → "com.acme.apis"
//   - Example: org="Acme-Corp" → "com.acme.corp.apis"
func DeriveMavenGroupId(org string) string {
	normalized := strings.ToLower(org)
	normalized = strings.ReplaceAll(normalized, "-", ".")
	return "com." + normalized + ".apis"
}

// DeriveMavenArtifactId computes the Maven artifactId for an API.
//
// Rules:
//   - Combines domain (if present), name, and line with hyphens
//   - Appends "-proto" suffix to distinguish schema artifacts
//   - Example: proto/payments/ledger/v1 → "payments-ledger-v1-proto"
//   - Example: proto/orders/v1 (3-part) → "orders-v1-proto"
func DeriveMavenArtifactId(api *APIIdentity) string {
	var parts []string
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	parts = append(parts, "proto")
	return strings.Join(parts, "-")
}

// DeriveMavenCoords returns the full Maven coordinate string (groupId:artifactId).
//
// Example: org="acme", proto/payments/ledger/v1 → "com.acme.apis:payments-ledger-v1-proto"
func DeriveMavenCoords(org string, api *APIIdentity) string {
	return DeriveMavenGroupId(org) + ":" + DeriveMavenArtifactId(api)
}

// DeriveJavaPackage computes the Java package name for an API.
//
// Rules:
//   - Pattern: com.<org>.apis.<domain>.<name>.<line>
//   - Lowercased, hyphens replaced with dots
//   - Example: org="acme", proto/payments/ledger/v1 → "com.acme.apis.payments.ledger.v1"
//   - Example: org="acme", proto/orders/v1 → "com.acme.apis.orders.v1"
func DeriveJavaPackage(org string, api *APIIdentity) string {
	normalized := strings.ToLower(org)
	normalized = strings.ReplaceAll(normalized, "-", ".")
	parts := []string{"com", normalized, "apis"}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	return strings.Join(parts, ".")
}

// DeriveNpmPackage computes the scoped npm package name for an API.
//
// Rules:
//   - Pattern: @<org>/<domain>-<name>-<line>-proto (4-part) or @<org>/<name>-<line>-proto (3-part)
//   - Lowercased, hyphens join path segments, -proto suffix
//   - Example: org="acme", proto/payments/ledger/v1 → "@acme/payments-ledger-v1-proto"
//   - Example: org="acme", proto/orders/v1 (3-part) → "@acme/orders-v1-proto"
func DeriveNpmPackage(org string, api *APIIdentity) string {
	scope := strings.ToLower(org)
	var parts []string
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	parts = append(parts, "proto")
	return "@" + scope + "/" + strings.Join(parts, "-")
}

// DeriveTsImport returns the TypeScript import path for an API.
// In TypeScript, the import path is the npm package name itself.
func DeriveTsImport(org string, api *APIIdentity) string {
	return DeriveNpmPackage(org, api)
}

// pep440PreRe matches SemVer pre-release tags: alpha, beta, rc with optional dot-separator.
var pep440PreRe = regexp.MustCompile(`-(alpha|beta|rc)\.?(\d+)`)

// NormalizePEP440Version converts a SemVer version string to PEP 440 format.
//
// Rules:
//   - Strips leading "v" prefix
//   - Converts -alpha.N → aN
//   - Converts -beta.N → bN
//   - Converts -rc.N → rcN
//   - Example: "v1.2.3" → "1.2.3"
//   - Example: "v1.0.0-beta.1" → "1.0.0b1"
//   - Example: "v1.0.0-alpha.2" → "1.0.0a2"
//   - Example: "v1.0.0-rc.1" → "1.0.0rc1"
func NormalizePEP440Version(semver string) string {
	v := strings.TrimPrefix(semver, "v")

	v = pep440PreRe.ReplaceAllStringFunc(v, func(match string) string {
		sub := pep440PreRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		tag, num := sub[1], sub[2]
		switch tag {
		case "alpha":
			return "a" + num
		case "beta":
			return "b" + num
		case "rc":
			return "rc" + num
		}
		return match
	})

	return v
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

// DeriveLanguageCoords fills complete language coordinates from an API identity
// and source repository. If importRoot is non-empty, it is used as the Go
// module prefix instead of sourceRepo.
func DeriveLanguageCoords(sourceRepo string, api *APIIdentity) (map[string]LanguageCoords, error) {
	return DeriveLanguageCoordsWithRoot(sourceRepo, "", "", api)
}

// DeriveLanguageCoordsWithRoot derives language coordinates for all supported
// languages. Parameters:
//   - sourceRepo: Git hosting path (e.g. "github.com/acme/apis")
//   - importRoot: optional Go-specific import root override
//   - org: organization name for Python package naming; if empty, Python coords are omitted
func DeriveLanguageCoordsWithRoot(sourceRepo, importRoot, org string, api *APIIdentity) (map[string]LanguageCoords, error) {
	goRoot := EffectiveGoRoot(sourceRepo, importRoot)
	goMod, err := DeriveGoModule(goRoot, api)
	if err != nil {
		return nil, fmt.Errorf("deriving Go module: %w", err)
	}
	goImport, err := DeriveGoImport(goRoot, api)
	if err != nil {
		return nil, fmt.Errorf("deriving Go import: %w", err)
	}

	coords := map[string]LanguageCoords{
		"go": {
			Module: goMod,
			Import: goImport,
		},
	}

	if org != "" {
		coords["python"] = LanguageCoords{
			Module: DerivePythonDistName(org, api),
			Import: DerivePythonImport(org, api),
		}
		coords["java"] = LanguageCoords{
			Module: DeriveMavenCoords(org, api),
			Import: DeriveJavaPackage(org, api),
		}
		coords["typescript"] = LanguageCoords{
			Module: DeriveNpmPackage(org, api),
			Import: DeriveTsImport(org, api),
		}
	}

	return coords, nil
}

// BuildIdentityBlock creates a full identity section from an API ID string
// and source repository. This is the primary entry point for populating
// identity fields from minimal inputs.
func BuildIdentityBlock(apiID, sourceRepo, lifecycle, currentVersion string) (*APIIdentity, *SourceIdentity, *ReleaseInfo, map[string]LanguageCoords, error) {
	return BuildIdentityBlockWithRoot(apiID, sourceRepo, "", "", lifecycle, currentVersion)
}

// BuildIdentityBlockWithRoot is like BuildIdentityBlock but accepts an
// optional importRoot (Go import root override) and org (organization name
// for Python package naming).
func BuildIdentityBlockWithRoot(apiID, sourceRepo, importRoot, org, lifecycle, currentVersion string) (*APIIdentity, *SourceIdentity, *ReleaseInfo, map[string]LanguageCoords, error) {
	api, err := ParseAPIID(apiID)
	if err != nil {
		return nil, nil, nil, nil, err
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

	langs, err := DeriveLanguageCoordsWithRoot(sourceRepo, importRoot, org, api)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return api, source, release, langs, nil
}

// FormatIdentityReport produces a human-readable multi-line report
// of an API's canonical identity information.
func FormatIdentityReport(api *APIIdentity, source *SourceIdentity, release *ReleaseInfo, langs map[string]LanguageCoords) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("API:        %s\n", api.ID))
	sb.WriteString(fmt.Sprintf("Format:     %s\n", api.Format))
	sb.WriteString(fmt.Sprintf("Domain:     %s\n", api.Domain))
	sb.WriteString(fmt.Sprintf("Name:       %s\n", api.Name))
	sb.WriteString(fmt.Sprintf("Line:       %s\n", api.Line))

	if api.Lifecycle != "" {
		sb.WriteString(fmt.Sprintf("Lifecycle:  %s\n", api.Lifecycle))
	}

	if source != nil {
		sb.WriteString(fmt.Sprintf("Source:     %s/%s\n", source.Repo, source.Path))
	}

	if release != nil && release.Current != "" {
		sb.WriteString(fmt.Sprintf("Release:    %s\n", release.Current))
		sb.WriteString(fmt.Sprintf("Tag:        %s\n", DeriveTag(api.ID, release.Current)))
	}

	if goCoords, ok := langs["go"]; ok {
		sb.WriteString(fmt.Sprintf("Go module:  %s\n", goCoords.Module))
		sb.WriteString(fmt.Sprintf("Go import:  %s\n", goCoords.Import))
	}

	if pyCoords, ok := langs["python"]; ok {
		sb.WriteString(fmt.Sprintf("Py dist:    %s\n", pyCoords.Module))
		sb.WriteString(fmt.Sprintf("Py import:  %s\n", pyCoords.Import))
	}

	if javaCoords, ok := langs["java"]; ok {
		sb.WriteString(fmt.Sprintf("Maven:      %s\n", javaCoords.Module))
		sb.WriteString(fmt.Sprintf("Java pkg:   %s\n", javaCoords.Import))
	}

	if tsCoords, ok := langs["typescript"]; ok {
		sb.WriteString(fmt.Sprintf("npm:        %s\n", tsCoords.Module))
	}

	return sb.String()
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
