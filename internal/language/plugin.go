// Package language provides a plugin-based framework for multi-language support.
//
// Each language (Go, Python, Java, TypeScript) implements the LanguagePlugin
// interface and registers itself via init(). The core framework iterates
// registered plugins for identity derivation, display formatting, unlink hints,
// code generation hooks, and documentation contribution.
//
// To add a new language, see CONTRIBUTING.md in this directory.
package language

import (
	"github.com/infobloxopen/apx/internal/config"
)

// DerivationContext provides the inputs needed to derive language-specific
// coordinates for an API identity.
type DerivationContext struct {
	// SourceRepo is the Git hosting path (e.g. "github.com/acme/apis").
	SourceRepo string

	// ImportRoot is an optional Go-specific import root override
	// (e.g. "go.acme.dev/apis"). If empty, SourceRepo is used.
	ImportRoot string

	// Org is the organization name. When empty, org-dependent languages
	// (Python, Java, TypeScript) are skipped.
	Org string

	// API is the parsed API identity.
	API *config.APIIdentity
}

// ReportLine represents a single line in a human-readable identity report.
type ReportLine struct {
	Label string // e.g. "Go module", "Py dist", "Maven", "npm"
	Value string // the derived coordinate value
}

// UnlinkHint provides post-unlink guidance for switching to a released package.
type UnlinkHint struct {
	Message string // e.g. "Go: Run 'go get ...' to add released module"
}

// LanguagePlugin defines the contract for a language support plugin.
// All built-in languages implement this interface and register via init().
type LanguagePlugin interface {
	// Name returns the canonical key for this language (e.g. "go", "python", "java", "typescript").
	Name() string

	// Tier returns the display priority. Tier 1 (Go) is always shown first;
	// Tier 2 languages are shown alphabetically.
	Tier() int

	// Available returns true if this plugin can derive coordinates given the context.
	// For example, Python/Java/TypeScript require Org != "".
	Available(ctx DerivationContext) bool

	// DeriveCoords computes the language-specific module and import coordinates.
	DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error)

	// ReportLines returns display-formatted lines for an identity report.
	ReportLines(coords config.LanguageCoords) []ReportLine

	// UnlinkHint returns the post-unlink guidance message, or nil if none.
	UnlinkHint(ctx DerivationContext) *UnlinkHint
}

// Scaffolder is an optional interface for plugins that scaffold output
// during code generation (e.g. Python pyproject.toml).
type Scaffolder interface {
	Scaffold(overlayPath string, ctx DerivationContext) error
}

// PostGenHook is an optional interface for plugins that run a hook after
// code generation (e.g. Go go.work sync).
type PostGenHook interface {
	PostGen(workDir string) error
}

// Linker is an optional interface for plugins that support activating local
// overlays into the language's package manager (e.g. Python pip install -e).
type Linker interface {
	Link(workDir, filterPath string) error
}

// Unlinker is an optional interface for plugins that support deactivating local
// overlays from the language's package manager (e.g. Python pip uninstall).
type Unlinker interface {
	Unlink(workDir, filterPath string) error
}

// DocContributor is an optional interface for plugins that contribute
// documentation fragments for build-time doc generation.
type DocContributor interface {
	DocMeta() DocMeta
}

// DocMeta contains structured metadata and prose fragments for doc generation.
type DocMeta struct {
	// SupportMatrix provides one row for the Language Support Matrix table.
	// Keys: "published_artifact", "local_overlay", "resolution", "codegen",
	//        "dev_command", "unlink_hint", "tier"
	SupportMatrix map[string]string

	// IdentityRows provides rows for the Identity Derivation table.
	IdentityRows []IdentityRow

	// PathMappings provides example rows for path mapping tables.
	PathMappings []PathMapping

	// Sections contains named markdown fragments for prose docs.
	// Standard keys: "code_generation", "dev_workflow", "troubleshooting"
	Sections map[string]string
}

// IdentityRow represents one row in an identity derivation table.
type IdentityRow struct {
	CoordType    string // e.g. "Module", "Import", "Maven coords"
	DerivedValue string // e.g. "github.com/acme/apis/proto/payments/ledger"
}

// PathMapping represents one example row in a path mapping table.
type PathMapping struct {
	APXPath     string // e.g. "proto/payments/ledger/v1"
	TargetCoord string // e.g. "github.com/<org>/apis/proto/payments/ledger/v1"
	Description string // column header context
}
