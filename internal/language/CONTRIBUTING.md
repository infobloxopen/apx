# Adding a New Language to APX

This guide walks through every step needed to add a new language plugin
to APX. The plugin system lives in `internal/language/` — follow this
checklist mechanically and the new language will be automatically wired
into identity derivation, CLI output, unlink hints, doc generation, and
the release manifest.

## Prerequisites

- Understand the APX API identity model (format/domain/name/line)
- Know how the target language's package ecosystem works (naming, imports, registries)

## Step-by-Step Checklist

### 1. Identity Derivation Functions

**File:** `internal/config/identity.go`

Add two functions for the new language:

```go
func Derive<Lang>Module(org string, api *APIIdentity) string { ... }
func Derive<Lang>Import(org string, api *APIIdentity) string { ... }
```

**Tests:** Add corresponding test cases in `internal/config/identity_test.go`.

### 2. Plugin Implementation

**File:** `internal/language/<lang>.go`

Implement the `LanguagePlugin` interface:

```go
func init() { Register(&<lang>Plugin{}) }

type <lang>Plugin struct{}

func (p *<lang>Plugin) Name() string                                              // canonical key: "go", "python", etc.
func (p *<lang>Plugin) Tier() int                                                 // 1=Go (always first), 2=others
func (p *<lang>Plugin) Available(ctx DerivationContext) bool                       // usually: ctx.Org != ""
func (p *<lang>Plugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error)
func (p *<lang>Plugin) ReportLines(coords config.LanguageCoords) []ReportLine
func (p *<lang>Plugin) UnlinkHint(ctx DerivationContext) *UnlinkHint
```

**Optionally implement:**

- `Scaffolder` — if the language needs scaffolding during `apx gen` (e.g. pyproject.toml)
- `PostGenHook` — if the language needs a post-generation step (e.g. go.work sync)
- `Linker` — if the language supports `apx link` (e.g. pip install -e)

**Tests:** `internal/language/<lang>_test.go` with at minimum:
- `TestName`, `TestAvailable`, `TestDeriveCoords`, `TestReportLines`, `TestUnlinkHint`
- If implementing optional interfaces: test that the type assertion succeeds

**Template:** Copy `internal/language/typescript.go` as a starting point — it's
the simplest Tier 2 plugin with all required methods.

### 3. Documentation Fragments

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

### 4. Testscript (Integration Test)

**File:** `testdata/script/<lang>_identity.txt`

Test `apx inspect identity` shows correct coordinates:

```
exec apx inspect identity proto/payments/ledger/v1
stdout '<expected coordinate>'

exec apx --json inspect identity proto/payments/ledger/v1
stdout '"<lang>"'
```

Look at `testdata/script/typescript_identity.txt` or `java_identity.txt`
as a template.

### 5. Generate and Verify

```bash
GOTOOLCHAIN=go1.26.1 go generate ./internal/language/...  # regenerates doc includes
GOTOOLCHAIN=go1.26.1 go build ./...                        # verify compilation
GOTOOLCHAIN=go1.26.1 go test ./internal/language/ -v        # plugin tests
GOTOOLCHAIN=go1.26.1 go test ./internal/config/ -v          # derivation tests
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
| `internal/config/identity.go` | Derive functions (building blocks) |
| `internal/language/<lang>.go` | Plugin implementation |
| `internal/language/<lang>_test.go` | Plugin unit tests |
| `internal/language/<lang>_doc.go` | DocContributor + embeds |
| `internal/language/<lang>_doc/*.md` | Documentation fragments |
| `testdata/script/<lang>_identity.txt` | Integration test |
