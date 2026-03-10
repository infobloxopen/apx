# Initialization Guide

APX provides intelligent initialization for both **canonical repositories** (organization-wide API source of truth) and **app repositories** (where teams author schemas). The system detects your context, suggests smart defaults, and guides you through setup with context-aware prompts.

## Overview

APX supports two distinct initialization patterns:

<div class="grid cards" markdown>
-   **Canonical Repository**

    ---

    - Single source of truth for all org APIs
    - `github.com/<org>/apis` structure
    - Policy enforcement and governance
    - CI-only releases and tagging

-   **App Repository**

    ---

    - Daily schema authoring by teams
    - Releases to canonical via PRs
    - Local development and testing
    - Tag-based release workflow

</div>

Key benefits:
- **🎯 Context-aware**: Detects canonical vs app repo setup
- **🤖 Smart defaults**: Environment detection and path suggestions
- **📋 Guided workflows**: Step-by-step prompts for each repo type
- **🔧 Flexible modes**: Interactive, partial, and non-interactive

## Initialization Modes

### Canonical Repository Setup

Initialize the organization-wide API repository:

```bash
# Full interactive setup
apx init canonical

# With organization and repo specified
apx init canonical --org=mycompany --repo=apis

# Non-interactive with all flags
apx init canonical --org=mycompany --repo=apis --non-interactive
```

This creates the canonical structure:
```
apis/
├── buf.yaml                 # org-wide policy
├── buf.work.yaml            # workspace config  
├── CODEOWNERS              # per-path ownership
├── catalog/
│  ├── .gitignore           # ignores generated catalog.yaml
│  └── Dockerfile           # scratch-based image with OCI labels
└── proto/                  # (+ other format dirs)
```

### App Repository Setup

Initialize schema authoring in an app repository:

```bash
# Full interactive - detects best practices
apx init app

# Specify path interactively
apx init app internal/apis/proto/payments/ledger

# Non-interactive with full configuration
apx init app --org=mycompany --repo=myapp --non-interactive \
  internal/apis/proto/payments/ledger
```

### Schema Type Detection

When initializing app repos, APX guides you through schema selection:

1. **Repository Purpose**: Canonical (org-wide) vs App (team authoring)
2. **Schema Format**: proto, openapi, avro, jsonschema, parquet  
3. **API Path**: Internal path and canonical destination mapping
4. **Languages**: Target languages for code generation

## Smart Defaults Detection

APX intelligently detects your context and suggests appropriate configurations:

### Repository Type Detection

- **Canonical patterns**: Detects repository names like `apis`, `schemas`, `platform-apis`
- **App repo patterns**: Detects service/application repositories with existing code
- **Path analysis**: Examines directory structure for API authoring patterns

### Organization & Path Detection

- **GitHub paths**: Extracts from paths like `/Users/dev/github.com/mycompany/service`
- **Git remotes**: Parses `origin` remote URLs for organization info
- **CODEOWNERS**: Reads existing ownership patterns for suggestions

### Import Root Detection

When running `apx init app`, APX resolves `import_root` automatically:

1. Fetches the canonical repo's `apx.yaml` via GitHub (public or private)
2. Falls back to the locally cached catalog (`~/.cache/apx/catalogs/{org}/{repo}/`)
3. Pre-fills the interactive prompt with the detected value

This means you only need to set `import_root` once in the canonical repo — all `apx init app` runs inherit it.

### Schema Format Detection

- **Proto focus**: Defaults to Protocol Buffers for new projects
- **Existing schemas**: Detects current schema files in repository
- **Build files**: Analyzes `buf.yaml`, `openapi.yaml`, etc.

### Language & Tooling Detection

APX scans for project indicators:

| File | Detected Language | Generated Code |
|------|------------------|----------------|
| `go.mod` | Go | `*.pb.go`, `*_grpc.pb.go` |
| `requirements.txt`, `pyproject.toml` | Python | `*_pb2.py` |
| `pom.xml`, `build.gradle` | Java | `*.java` |
| `package.json` | TypeScript/JavaScript | `*.ts`, `*.js` |

## Interactive Prompts

### Repository Type Selection

```
? What type of repository are you setting up?
❯ canonical - Organization-wide API source of truth (github.com/org/apis)
  app - Team repository for authoring schemas (releases to canonical)
```

### Canonical Repository Prompts

```
? Organization name: (detected: mycompany)
? Canonical repository path: github.com/mycompany/apis
? Initial schema formats to support:
  ☑ proto - Protocol Buffers (recommended)
  ☐ openapi - OpenAPI specifications  
  ☐ avro - Apache Avro schemas
  ☐ jsonschema - JSON Schema definitions
  ☐ parquet - Parquet schemas
```

### App Repository Prompts

```
? Schema format for this API: (Use arrow keys)
❯ proto - Protocol Buffers for gRPC services
  openapi - OpenAPI specifications for REST APIs
  avro - Apache Avro for data serialization
  jsonschema - JSON Schema for validation
  parquet - Apache Parquet for analytics

? API path in your app repo:
  internal/apis/proto/payments/ledger/v1

? Canonical destination path:  
  proto/payments/ledger/v1

? Canonical repository: github.com/mycompany/apis
? Target languages: [go, python, java]
```

### Configuration Summary

APX shows a comprehensive summary before creating files:

```
📋 App Repository Configuration:
   Schema Format: proto
   Local Path: internal/apis/proto/payments/ledger/v1
   Canonical Path: proto/payments/ledger/v1
   Canonical Repo: github.com/mycompany/apis
   
📋 Code Generation:
   Output Directory: internal/gen
   Languages: go, python
   
? Proceed with this configuration? (Y/n)
```

## Command-Line Flags

Override detected defaults and skip prompts with flags:

### Canonical Repository Flags

- `--org VALUE`: Organization name for canonical repo
- `--repo VALUE`: Repository name
- `--skip-git`: Skip git initialization
- `--non-interactive`: Use all defaults without prompts

### App Repository Flags

- `--org VALUE`: Organization name
- `--repo VALUE`: Repository name for the app
- `--non-interactive`: Skip all prompts (requires `--org` and `--repo`)

### Examples

#### Canonical Repository Setup
```bash
# Interactive with organization override
apx init canonical --org "acme-corp"

# Full non-interactive setup
apx init canonical --non-interactive \
  --org "mycompany" \
  --formats "proto,openapi" \
  github.com/mycompany/apis
```

#### App Repository Setup
```bash
# Interactive with canonical repo specified
apx init app --canonical-repo "github.com/mycompany/apis" \
  internal/apis/proto/payments/ledger/v1

# Complete non-interactive setup  
apx init app --non-interactive \
  --canonical-repo "github.com/mycompany/apis" \
  --canonical-path "proto/payments/ledger/v1" \
  --languages "go,python" \
  internal/apis/proto/payments/ledger/v1
```

## Generated Structures

`apx init` creates different directory structures based on repository type. For the full breakdown of each:

- **Canonical repos**: See [Canonical Repo Setup](../canonical-repo/setup.md) for the complete generated structure, CI workflows, and GitHub protection configuration
- **App repos**: See [App Repo Layout](../app-repos/layout.md) for the complete file-by-file reference

## Best Practices

### Canonical Repository Setup

1. **Single canonical repo**: One `github.com/<org>/apis` for entire organization
2. **Format separation**: Separate top-level directories (proto/, openapi/, etc.)
3. **Domain organization**: Group APIs by business domain (payments/, users/)
4. **Protection rules**: Protect main branch and tag patterns (`proto/**/v*`)
5. **CODEOWNERS**: Assign clear ownership per API path

### App Repository Setup

1. **Internal directory**: Use `internal/apis/` to prevent vendoring
2. **Version directories**: Separate v1/, v2/ for major version evolution
3. **No local go.mod**: Let APX synthesize canonical go.mod on release
4. **Buf workspace**: Include all version directories in `buf.work.yaml`
5. **Generated code**: Never commit `internal/gen/` - use `.gitignore`

### Naming Conventions

**Protocol Buffer packages**:
```protobuf
// Include organization and version
package myorg.payments.ledger.v1;

// Go package path (canonical repo)
option go_package = "github.com/myorg/apis/proto/payments/ledger/v1";
```

**Directory structure**:
```
# App repo path (with internal/)
internal/apis/proto/payments/ledger/v1/

# Canonical repo path (no internal/)  
proto/payments/ledger/v1/
```

### Team Workflows

1. **Start with app repos**: Teams author in familiar repositories
2. **Tag-based releases**: Use `proto/domain/api/v1/v1.2.3` tag format
3. **CI automation**: Let CI handle canonical repo releasing
4. **Review process**: CODEOWNERS approval for all canonical changes

## Troubleshooting

### Interactive Mode Not Working

Interactive mode is automatically disabled in:
- CI environments (`CI=true`)
- Non-TTY terminals (`TERM=dumb`)
- When explicitly disabled (`--non-interactive`)

### Detection Issues

If smart defaults aren't working:
1. Ensure you're in the correct directory
2. Check file permissions on project files (`go.mod`, etc.)
3. Use explicit flags to override detection

### Flag Order Issues

Due to CLI parsing, flags must come before arguments:

```bash
# ✅ Correct
apx init --non-interactive proto service.v1

# ❌ Incorrect  
apx init proto service.v1 --non-interactive
```

## Integration Examples

### Setting Up Canonical Repository

```bash
# Organization admin sets up canonical repo
git clone https://github.com/mycompany/apis.git
cd apis
apx init canonical --org "mycompany"

# Configure protection rules and CI
git add .
git commit -m "Initialize canonical API repository"
git push
```

### Team Adding New API

```bash
# Team working in their service repository
cd ~/projects/payment-service
apx init app internal/apis/proto/payments/ledger/v1

# Start authoring
vim internal/apis/proto/payments/ledger/v1/ledger.proto
apx lint && apx gen go
```

### CI/CD Integration

For complete CI/CD workflow examples, see:

- [Tutorial: CI/CD Patterns](tutorial.md#cicd-patterns) — app repo and canonical repo workflow YAML
- [App Repos: CI Integration](../app-repos/ci-integration.md) — detailed app repo CI configuration
- [Canonical Repo: CI Templates](../canonical-repo/ci-templates.md) — canonical repo CI templates

### Team Onboarding Script

```bash
#!/bin/bash
# setup-api-authoring.sh

echo "🚀 Setting up API authoring in your service repository..."

# Check if we're in a service repo
if [[ ! -f "go.mod" ]] && [[ ! -f "package.json" ]] && [[ ! -f "pom.xml" ]]; then
  echo "⚠️  Run this script in your service repository"
  exit 1
fi

# Interactive setup
apx init app --canonical-repo="github.com/mycompany/apis"

echo "✅ API authoring setup complete!"
echo ""  
echo "Next steps:"
echo "1. Edit your schema files in internal/apis/"
echo "2. Run 'apx lint' to validate"
echo "3. Run 'apx gen go' to generate code"
echo "4. Tag releases with 'proto/domain/api/v1/v1.0.0'"
```

---

The initialization system supports both newcomers and experienced teams, adapting to your organization's canonical repo pattern while maintaining familiar development workflows.