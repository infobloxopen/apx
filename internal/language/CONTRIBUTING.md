# Adding a New Language to APX

This guide walks through every step needed to add a new language plugin
to APX. The plugin system lives in `internal/language/` — follow this
checklist mechanically and the new language will be automatically wired
into identity derivation, CLI output, unlink hints, doc generation, and
the release manifest.

## Prerequisites

- Understand the APX API identity model (format/domain/name/line)
- Know how the target language's package ecosystem works (naming, imports, registries)

## Org Name Normalization

The `org` value may contain hyphens (`acme-corp`) or mixed case (`Acme-Corp`).
Your derivation functions **must** handle this correctly:

- **Package manager names** (crate, dist, npm scope) — lowercase only; hyphens are usually valid
- **Language identifiers** (imports, modules, namespaces) — hyphens are typically invalid; replace with underscores or dots per the target language's rules

See existing plugins for patterns: Python (`_apis` namespace), Rust (`_` join),
C++ (`_` replace), Java (hyphens → dots). Tests must include a `"hyphenated org"` case.

## Step-by-Step Checklist

### 1. Plugin Implementation (with Derivation)

**File:** `internal/language/<lang>.go`

Each plugin is self-contained — derivation logic lives inside the plugin
as **unexported** functions, not in a shared module.

```go
func init() { Register(&<lang>Plugin{}) }

type <lang>Plugin struct{}

func (p *<lang>Plugin) Name() string                                              // canonical key: "go", "python", etc.
func (p *<lang>Plugin) Tier() int                                                 // 1=Go (always first), 2=others
func (p *<lang>Plugin) Available(ctx DerivationContext) bool                       // usually: ctx.Org != ""
func (p *<lang>Plugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error)
func (p *<lang>Plugin) ReportLines(coords config.LanguageCoords) []ReportLine
func (p *<lang>Plugin) UnlinkHint(ctx DerivationContext) *UnlinkHint

// Private derivation functions — called only by DeriveCoords:
func derive<Lang>Module(org string, api *config.APIIdentity) string { ... }
func derive<Lang>Import(org string, api *config.APIIdentity) string { ... }
```

Derivation functions (the language-specific coordinate derivation) are
**private to the plugin file**. They are never exported or placed in
`internal/config/`. The plugin's `DeriveCoords` method calls them and
returns a `config.LanguageCoords`.

**Optionally implement:**

- `Scaffolder` — if the language needs scaffolding during `apx gen` (e.g. pyproject.toml)
- `PostGenHook` — if the language needs a post-generation step (e.g. go.work sync)
- `Linker` — if the language supports `apx link` (e.g. pip install -e)

**Tests:** `internal/language/<lang>_test.go` with at minimum:
- `TestName`, `TestAvailable`, `TestDeriveCoords`, `TestReportLines`, `TestUnlinkHint`
- Derivation unit tests (e.g. `TestDerive<Lang>Module`, `TestDerive<Lang>Import`)
- If implementing optional interfaces: test that the type assertion succeeds

**Template:** Copy `internal/language/cpp.go` or `internal/language/rust.go`
as a starting point — they're the simplest Tier 2 plugins with self-contained
derivation.

### 2. Documentation Fragments

**Files:**
- `internal/language/<lang>_doc.go` — implements `DocContributor` with `//go:embed`
- `internal/language/<lang>_doc/code_generation.md` — code generation section
- `internal/language/<lang>_doc/dev_workflow.md` — development workflow section

The `_doc.go` file should look like:

```go
package language

import _ "embed"

//go:embed <lang>_doc/code_generation.md
var <lang>CodeGenDoc string

//go:embed <lang>_doc/dev_workflow.md
var <lang>DevWorkflowDoc string

func (p *<lang>Plugin) DocMeta() DocMeta {
    return DocMeta{
        SupportMatrix: map[string]string{
            "published_artifact": "...",
            "local_overlay":     "...",
            "resolution":        "...",
            "codegen":           "...",
            "dev_command":       "...",
            "unlink_hint":       "...",
            "tier":              "Tier 2",
        },
        IdentityRows: []IdentityRow{ ... },
        PathMappings: []PathMapping{ ... },
        Sections: map[string]string{
            "code_generation": <lang>CodeGenDoc,
            "dev_workflow":    <lang>DevWorkflowDoc,
        },
    }
}
```

### 3. Testscript (Integration Test)

**File:** `testdata/script/<lang>_identity.txt`

Test `apx inspect identity` shows correct coordinates:

```
exec apx inspect identity proto/payments/ledger/v1
stdout '<expected coordinate>'

exec apx --json inspect identity proto/payments/ledger/v1
stdout '"<lang>"'
```

Look at `testdata/script/cpp_identity.txt` or `rust_identity.txt`
as a template.

### 4. Generate and Verify

```bash
GOTOOLCHAIN=go1.26.1 go generate ./internal/language/...  # regenerates doc includes
GOTOOLCHAIN=go1.26.1 go build ./...                        # verify compilation
GOTOOLCHAIN=go1.26.1 go test ./internal/language/ -v        # plugin tests
GOTOOLCHAIN=go1.26.1 go test ./internal/config/ -v          # shared config tests
GOTOOLCHAIN=go1.26.1 go test ./... -count=1                 # full suite
GOTOOLCHAIN=go1.26.1 go test . -run TestScript -v           # all testscripts
ls docs/_generated/                                          # verify generated includes exist
```

## What Happens Automatically

When you register a new plugin via `init()`, the following are automatically
handled by the framework:

- **Identity derivation**: `language.DeriveAllCoords` includes the new language
- **CLI display**: `apx inspect identity`, `apx show`, reports all include the new coords
- **Manifest/record**: Release manifests include the new language in the `languages:` map
- **Unlink hints**: `apx unlink` prints the new language's hint
- **Gen scaffolding**: If `Scaffolder` is implemented, `apx gen <lang>` scaffolds automatically
- **Link support**: If `Linker` is implemented, `apx link <lang>` works automatically
- **Help text**: `apx gen` and `apx link` help text auto-includes the new language
- **Doc generation**: `cmd/docgen` assembles the new language's doc fragments

## File Summary

| File | Purpose |
|------|---------|
| `internal/language/<lang>.go` | Plugin + private derive functions |
| `internal/language/<lang>_test.go` | Plugin unit tests |
| `internal/language/<lang>_doc.go` | DocContributor + embeds |
| `internal/language/<lang>_doc/*.md` | Documentation fragments |
| `testdata/script/<lang>_identity.txt` | Integration test |
