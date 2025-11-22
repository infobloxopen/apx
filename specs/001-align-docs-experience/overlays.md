# Go Workspace Overlays: Design & Implementation

**Feature**: `001-align-docs-experience`  
**Created**: 2025-11-22  
**Status**: Implementation Complete

## Overview

**Go workspace overlays** are a development-time mechanism that allows applications to import canonical API module paths while transparently resolving them to locally generated code. This enables **zero-friction transitions** from local development to published module consumption without changing any import statements.

## Problem Statement

When developing microservices that consume API schemas (Protocol Buffers, OpenAPI, etc.), teams face a common challenge:

### Without Overlays (Traditional Approach)

```go
// During local development
import ledgerv1 "github.com/mycompany/my-service/internal/gen/go/proto/payments/ledger/v1"

// After schema is published, imports must be rewritten
import ledgerv1 "github.com/mycompany/apis-go/proto/payments/ledger/v1"
```

**Problems**:
- Import paths change when transitioning from local to published modules
- Requires find/replace across codebase when switching
- Breaks during transition period
- Temporary `replace` directives pollute `go.mod`
- Different import paths for local vs published create confusion

### With Overlays (APX Approach)

```go
// SAME import path works for both local development AND published consumption
import ledgerv1 "github.com/mycompany/apis-go/proto/payments/ledger/v1"

// During development: resolved to ./internal/gen/go/proto/payments/ledger@v1.2.3 via go.work
// After publishing:   resolved to published module github.com/mycompany/apis-go/proto/payments/ledger@v1.2.3
```

**Benefits**:
- **One import path forever** - use canonical paths from day one
- **Seamless transitions** - switch from local to published with zero code changes
- **No replace directives** - clean `go.mod` files
- **Portability** - same code works across development stages

## How It Works

### 1. Overlay Directory Structure

APX generates code into versioned overlay directories under `internal/gen/`:

```
your-service/
├── go.mod                                    # module github.com/mycompany/my-service
├── go.work                                   # managed by APX - contains overlay mappings
├── internal/
│   ├── gen/                                  # ALL generated code (git-ignored)
│   │   ├── go/proto/payments/ledger@v1.2.3/ # overlay for payments ledger v1.2.3
│   │   │   ├── go.mod                       # module github.com/myorg/apis-go/proto/payments/ledger
│   │   │   └── v1/
│   │   │       ├── ledger.pb.go             # generated protobuf code
│   │   │       └── ledger_grpc.pb.go        # generated gRPC stubs
│   │   ├── go/proto/users/profile@v1.0.1/   # overlay for users profile v1.0.1
│   │   │   ├── go.mod                       # module github.com/myorg/apis-go/proto/users/profile
│   │   │   └── v1/
│   │   │       └── profile.pb.go
│   │   ├── python/proto/payments/ledger/    # Python overlays (language-specific subdir)
│   │   └── java/proto/users/profile/        # Java overlays (language-specific subdir)
│   └── service/
│       └── payment_service.go               # your application code
└── main.go
```

**Key Structure Rules**:
- **Go overlays**: `internal/gen/go/{modulePath}@{version}/` (no language subdir)
- **Other languages**: `internal/gen/{language}/{modulePath}/` (language subdir required)
- **Module path**: Matches canonical import path (e.g., `proto/payments/ledger`)
- **Version tag**: Pinned version from `apx.lock` (e.g., `@v1.2.3`)
- **go.mod per overlay**: Each overlay is a separate Go module with canonical module path

### 2. Go Workspace File (go.work)

APX manages `go.work` to map canonical import paths to local overlay directories:

```go
go 1.24

use (
    .                                                      // your application module
    ./internal/gen/go/proto/payments/ledger@v1.2.3        // overlay 1
    ./internal/gen/go/proto/users/profile@v1.0.1          // overlay 2
)
```

**How Go Resolves Imports**:
1. Application imports `github.com/myorg/apis-go/proto/payments/ledger/v1`
2. Go checks `go.work` and finds `use ./internal/gen/go/proto/payments/ledger@v1.2.3`
3. Go reads `./internal/gen/go/proto/payments/ledger@v1.2.3/go.mod`:
   ```go
   module github.com/myorg/apis-go/proto/payments/ledger
   ```
4. Go resolves the import to the local overlay directory
5. Application compiles using locally generated code

### 3. Overlay Lifecycle

#### Create Overlays (`apx gen`)

```bash
# Generate Go client code for all dependencies in apx.lock
apx gen go
```

**What happens**:
1. APX reads `apx.lock` to find pinned dependencies:
   ```yaml
   dependencies:
     proto/payments/ledger/v1:
       repo: github.com/myorg/apis
       ref: proto/payments/ledger/v1.2.3
   ```
2. For each dependency, APX:
   - Creates overlay directory: `internal/gen/go/proto/payments/ledger@v1.2.3/`
   - Fetches schema from canonical repo at pinned version
   - Runs code generator (protoc, openapi-generator, etc.)
   - Creates `go.mod` with canonical module path
   - Generates code into overlay directory

#### Sync Workspace (`apx sync`)

```bash
# Update go.work to include all overlays
apx sync
```

**What happens**:
1. APX scans `internal/gen/` directory tree
2. Identifies all overlay directories (leaf directories or language-specific subdirs)
3. Filters for Go overlays (excludes `/python`, `/java` subdirs)
4. Rebuilds `go.work` with:
   - `go 1.24` directive
   - `use .` for application module
   - `use ./internal/gen/go/{path}` for each Go overlay
5. Writes updated `go.work` file

**When to run**:
- After `apx gen` (usually automatic)
- After `apx add` (adding new dependency)
- After `apx update` (updating dependency version)
- When overlays are out of sync with workspace

#### Remove Overlays (`apx unlink`)

```bash
# Remove overlay and switch to published module
apx unlink proto/payments/ledger/v1
```

**What happens**:
1. APX removes overlay directory: `rm -rf internal/gen/go/proto/payments/ledger@v1.2.3/`
2. APX runs `apx sync` to regenerate `go.work` (excluding removed overlay)
3. APX prompts user to fetch published module:
   ```
   Overlay removed. Run:
     go get github.com/myorg/apis-go/proto/payments/ledger@v1.2.3
   ```
4. User runs `go get` to add published module to `go.mod`
5. Application now uses published module instead of local overlay

**Result**: Import statements unchanged, resolution switches from local to published.

## Implementation Details

### Overlay Detection Logic

The `overlay.Manager.List()` function identifies overlays using this algorithm:

```go
func (m *Manager) List() ([]Overlay, error) {
    // Walk internal/gen/ tree
    filepath.Walk(m.overlayDir, func(path string, info os.FileInfo, err error) error {
        // Skip non-directories
        if !info.IsDir() { return nil }
        
        // Check if this is a language-specific overlay
        isLanguageOverlay := strings.HasSuffix(relPath, "/python") || 
                             strings.HasSuffix(relPath, "/java")
        
        // Check if this directory has subdirectories (excluding language dirs)
        hasSubdirs := hasNonLanguageSubdirectories(path)
        
        // This is an overlay if:
        // 1. It's a language-specific overlay (ends with /python or /java), OR
        // 2. It's a leaf directory (no subdirectories except language dirs)
        isOverlay := isLanguageOverlay || !hasSubdirs
        
        if isOverlay {
            // Extract module path and language
            // Add to overlay list
        }
    })
}
```

**Why This Logic?**

When creating `proto/payments/ledger/v1`, the file system has:
```
internal/gen/
├── go/
│   └── proto/           ← intermediate directory (NOT an overlay)
│       └── payments/    ← intermediate directory (NOT an overlay)
│           └── ledger@v1.2.3/  ← LEAF directory (IS an overlay)
├── python/
│   └── proto/           ← intermediate directory (NOT an overlay)
│       └── payments/    ← intermediate directory (NOT an overlay)
│           └── ledger/  ← intermediate directory (NOT an overlay)
│               └── python/  ← LANGUAGE directory (IS an overlay)
└── java/
    └── proto/
        └── payments/
            └── ledger/
                └── java/    ← LANGUAGE directory (IS an overlay)
```

The algorithm correctly identifies only the actual overlays, not intermediate path directories.

### go.work Generation

The `overlay.Manager.SyncWorkFile()` function:

```go
func (m *Manager) SyncWorkFile() error {
    // List all overlays
    overlays, err := m.List()
    
    // Build go.work content
    var content strings.Builder
    content.WriteString("go 1.24\n\n")
    content.WriteString("use (\n")
    content.WriteString("\t.\n")  // application module
    
    // Add each Go overlay
    for _, overlay := range overlays {
        if overlay.Language == "go" {
            relPath, _ := filepath.Rel(m.workspaceRoot, overlay.Path)
            content.WriteString(fmt.Sprintf("\t./%s\n", relPath))
        }
    }
    
    content.WriteString(")\n")
    
    // Write go.work
    os.WriteFile(workFilePath, []byte(content.String()), 0644)
}
```

**Key Features**:
- Only includes Go overlays (Python/Java use different resolution mechanisms)
- Uses relative paths from workspace root
- Regenerates entire file (idempotent)
- Maintains consistent formatting

### Overlay Struct

```go
// Overlay represents a generated code overlay
type Overlay struct {
    ModulePath string  // e.g., "proto/payments/ledger/v1"
    Language   string  // "go", "python", "java"
    Path       string  // absolute path to overlay directory
}
```

**Usage**:
```go
overlay := Overlay{
    ModulePath: "proto/payments/ledger/v1",
    Language:   "go",
    Path:       "/Users/dev/my-service/internal/gen/go/proto/payments/ledger@v1.2.3",
}
```

## Developer Workflows

### Workflow 1: Add New Dependency

```bash
# 1. Discover API in catalog
apx search payments ledger

# 2. Add dependency
apx add proto/payments/ledger/v1@v1.2.3
# → Updates apx.lock with pinned version

# 3. Generate client code
apx gen go
# → Creates internal/gen/go/proto/payments/ledger@v1.2.3/
# → Runs apx sync to update go.work

# 4. Write application code using canonical imports
cat > service.go <<EOF
import ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"

func createLedgerEntry() {
    client := ledgerv1.NewLedgerServiceClient(conn)
    // ... use client
}
EOF

# 5. Test locally
go test ./...
# → Imports resolve to local overlay via go.work
```

### Workflow 2: Update Dependency Version

```bash
# 1. Update to new version
apx update proto/payments/ledger/v1
# → Updates apx.lock to v1.2.4

# 2. Regenerate code
apx gen go && apx sync
# → Creates internal/gen/go/proto/payments/ledger@v1.2.4/
# → Updates go.work to point to new version
# → Removes old overlay

# 3. No code changes needed
go test ./...
# → Same imports, now resolve to v1.2.4 overlay
```

### Workflow 3: Transition to Published Module

```bash
# 1. Remove local overlay
apx unlink proto/payments/ledger/v1
# → Removes internal/gen/go/proto/payments/ledger@v1.2.3/
# → Regenerates go.work without this overlay

# 2. Fetch published module
go get github.com/myorg/apis-go/proto/payments/ledger@v1.2.3
# → Adds to go.mod dependencies

# 3. Verify (no code changes!)
go build ./...
# → Same imports now resolve to published module
```

## Comparison with Alternatives

### vs. Replace Directives

**Replace Directives** (`go.mod`):
```go
replace github.com/myorg/apis-go/proto/payments/ledger => ./internal/gen/go/proto/payments/ledger
```

**Problems**:
- Pollutes `go.mod` with local-only directives
- Easy to accidentally commit
- Requires cleanup when switching to published
- Doesn't support versioned overlays well

**Overlays** (`go.work`):
```go
use ./internal/gen/go/proto/payments/ledger@v1.2.3
```

**Benefits**:
- `go.work` is development-only (often git-ignored)
- Clean separation of concerns
- Supports multiple versions simultaneously
- APX manages automatically

### vs. Vendor Directory

**Vendor**:
- Commits generated code to repository
- Large repository size
- Merge conflicts on generated code
- Stale generated code

**Overlays**:
- Generated code never committed (in `.gitignore`)
- Small repository size
- No merge conflicts
- Always fresh from `apx gen`

### vs. Monorepo

**Monorepo**:
- All services and APIs in one repository
- Complex CI/CD
- Tight coupling

**Overlays + Canonical Repo**:
- Independent service repositories
- Simple per-service CI/CD
- Loose coupling via versioned APIs

## Best Practices

### 1. Always Ignore Generated Code

**`.gitignore`**:
```gitignore
/internal/gen/
```

**Why**: Generated code is reproducible from `apx.lock`. Never commit it.

### 2. Commit apx.lock

**Why**: Pins exact versions of dependencies and toolchains for reproducible builds.

### 3. Run apx sync After Dependency Changes

```bash
# After any of these commands:
apx add proto/payments/ledger/v1@v1.2.3
apx update proto/payments/ledger/v1
apx remove proto/payments/ledger/v1
apx gen go

# Always run:
apx sync
```

**Why**: Keeps `go.work` synchronized with actual overlay state.

### 4. Use Canonical Imports From Day One

**Good**:
```go
import ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"
```

**Bad**:
```go
import ledgerv1 "github.com/myorg/my-service/internal/gen/go/proto/payments/ledger/v1"
```

**Why**: Canonical imports work forever. Local paths break when transitioning to published.

### 5. Clean Overlays in CI

**CI Script**:
```bash
# Ensure clean state
rm -rf internal/gen/

# Regenerate from apx.lock
apx gen go
apx sync

# Build and test
go build ./...
go test ./...
```

**Why**: Prevents stale overlays from causing false positives.

## Troubleshooting

### Problem: Imports not resolving

**Symptom**:
```
package github.com/myorg/apis-go/proto/payments/ledger/v1: cannot find package
```

**Solutions**:
1. Check `go.work` exists and contains overlay:
   ```bash
   cat go.work | grep payments/ledger
   ```
2. Regenerate overlays:
   ```bash
   apx gen go && apx sync
   ```
3. Verify overlay directory exists:
   ```bash
   ls -la internal/gen/go/proto/payments/ledger@*/
   ```

### Problem: Multiple versions of same API

**Symptom**:
```
ambiguous import: multiple modules provide github.com/myorg/apis-go/proto/payments/ledger
```

**Solution**:
```bash
# List all overlays
ls -d internal/gen/go/proto/payments/ledger@*

# Remove all versions
rm -rf internal/gen/go/proto/payments/ledger@*

# Regenerate from apx.lock (single version)
apx gen go && apx sync
```

### Problem: go.work has stale overlays

**Symptom**:
```
go: directory ./internal/gen/go/proto/payments/ledger@v1.2.3 does not exist
```

**Solution**:
```bash
# Regenerate go.work from actual overlay directories
apx sync
```

### Problem: Accidentally committed internal/gen/

**Solution**:
```bash
# Remove from git
git rm -rf internal/gen/

# Add to .gitignore
echo "/internal/gen/" >> .gitignore

# Commit the fix
git add .gitignore
git commit -m "fix: remove generated code, add to gitignore"
```

## Future Enhancements

### Planned Features

1. **Language-Specific Overlay Managers**
   - Python: virtual environment overlays
   - Java: local Maven repository overlays
   - Rust: Cargo workspace overlays

2. **Overlay Caching**
   - Cache generated code by (modulePath, version, language)
   - Reuse cached overlays across projects
   - Faster `apx gen` for unchanged dependencies

3. **Overlay Garbage Collection**
   - Detect unused overlays (not in `apx.lock`)
   - Clean up automatically or with `apx clean`
   - Report disk space savings

4. **Overlay Verification**
   - Checksum generated code against canonical schemas
   - Detect manual modifications to overlays
   - Warn about drift

## References

- **Go Workspaces**: https://go.dev/doc/tutorial/workspaces
- **Canonical Import Paths**: `/docs/getting-started/quickstart.md`
- **APX Architecture**: `/specs/001-align-docs-experience/plan.md`
- **Code Generation**: `/docs/cli-reference/index.md#gen-command`
- **Publishing Workflow**: `/docs/publishing/index.md`

## Glossary

- **Overlay**: A local directory containing generated code that shadows a canonical module path
- **Canonical Import Path**: The permanent import path used in application code (e.g., `github.com/org/apis-go/proto/domain/api/v1`)
- **go.work**: Go workspace file that maps canonical paths to local overlay directories
- **Module Path**: The schema's path in the canonical repository (e.g., `proto/payments/ledger/v1`)
- **Overlay Directory**: Physical location of generated code (e.g., `internal/gen/go/proto/payments/ledger@v1.2.3/`)
