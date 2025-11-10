# Interactive Initialization

APX provides intelligent interactive initialization for both **canonical repositories** (organization-wide API source of truth) and **app repositories** (where teams author schemas). The system guides you through setup with smart defaults and context-aware prompts.

## Overview

APX supports two distinct initialization patterns:

::::{grid} 1 1 2 2
:gutter: 3

:::{grid-item-card} **Canonical Repository**
^^^
- Single source of truth for all org APIs
- `github.com/<org>/apis` structure
- Policy enforcement and governance
- CI-only releases and tagging
:::

:::{grid-item-card} **App Repository**  
^^^
- Daily schema authoring by teams
- Publishes to canonical via PRs
- Local development and testing
- Tag-based publishing workflow
:::

::::

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

# With organization specified
apx init canonical --org mycompany

# Non-interactive with full path
apx init canonical --non-interactive github.com/mycompany/apis
```

This creates the canonical structure:
```
apis/
├── buf.yaml                 # org-wide policy
├── buf.work.yaml            # workspace config  
├── CODEOWNERS              # per-path ownership
├── catalog/
│  └── catalog.yaml         # generated API index
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
apx init app --non-interactive \
  --canonical=github.com/mycompany/apis \
  internal/apis/proto/payments/ledger/v1
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
  app - Team repository for authoring schemas (publishes to canonical)
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
- `--formats VALUE`: Comma-separated schema formats (proto,openapi,avro)
- `--non-interactive`: Use all defaults without prompts

### App Repository Flags

- `--canonical-repo VALUE`: Target canonical repository URL
- `--canonical-path VALUE`: Destination path in canonical repo
- `--languages VALUE`: Target languages for code generation
- `--non-interactive`: Skip all prompts

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

APX creates different structures based on repository type:

### Canonical Repository Structure

```
apis/                           # github.com/org/apis
├── buf.yaml                    # org-wide lint/breaking policy
├── buf.work.yaml               # workspace config
├── CODEOWNERS                  # per-path ownership
├── .github/
│  └── workflows/
│     ├── validate.yml          # PR validation
│     └── release.yml           # tag creation & publishing
├── catalog/
│  └── catalog.yaml            # generated API index
├── proto/                     # Protocol Buffers
│  └── domain/
│     └── service/
│        ├── go.mod           # v1 module (no /v1 suffix)
│        ├── v1/
│        │  └── service.proto
│        └── v2/              # future major versions
├── openapi/                   # REST API specs
├── avro/                      # Event schemas  
├── jsonschema/                # Validation schemas
└── parquet/                   # Analytics schemas
```

### App Repository Structure

```
<app-repo>/                    # team's service repository
├── internal/
│  └── apis/
│     └── proto/              # or openapi/, avro/, etc.
│        └── payments/
│           └── ledger/
│              ├── v1/
│              │  └── ledger.proto
│              └── v2/        # future versions
├── buf.work.yaml             # Buf workspace
├── apx.yaml                 # APX configuration
├── apx.lock                  # pinned toolchain
├── .gitignore                # excludes internal/gen/
└── .github/
   └── workflows/
      └── publish-api.yml      # tag-based publishing
```

### Key Configuration Files

**Canonical apx.yaml**:
```yaml
# Minimal - mainly for CI tooling
project:
  type: canonical
  org: mycompany

validation:
  policy:
    banned_annotations: ["gorm.*", "database.*"]
```

**App apx.yaml**:
```yaml
apis:
  - kind: proto
    path: internal/apis/proto/payments/ledger/v1
    canonical: proto/payments/ledger/v1

codegen:
  out: internal/gen
  languages: [go, python]

publishing:
  canonical_repo: github.com/mycompany/apis
  strategy: subtree
```

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
3. **No local go.mod**: Let APX synthesize canonical go.mod on publish
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
3. **CI automation**: Let CI handle canonical repo publishing
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

#### App Repository CI (Publishing)
```yaml
# .github/workflows/publish-api.yml
name: Publish API
on:
  push:
    tags: ['proto/*/*/v*/v*']

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - run: apx fetch --ci
      - run: apx lint && apx breaking
      - run: apx publish --module-path=internal/apis/${GITHUB_REF_NAME%/v*} \
               --canonical-repo=github.com/mycompany/apis
```

#### Canonical Repository CI (Validation & Release)
```yaml
# .github/workflows/validate-release.yml  
name: Validate + Release
on:
  pull_request:
    paths: ['proto/**', 'openapi/**']
  push:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: apx fetch --ci && apx lint && apx breaking

  release:
    if: github.ref == 'refs/heads/main'
    needs: [validate]
    runs-on: ubuntu-latest
    permissions: { contents: write }
    steps:
      - uses: actions/checkout@v4
      - run: apx version verify && apx tag subdir && apx packages publish
```

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

The interactive initialization system supports both newcomers and experienced teams, adapting to your organization's canonical repo pattern while maintaining familiar development workflows.