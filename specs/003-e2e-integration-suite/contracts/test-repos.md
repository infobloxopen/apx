# Test Repository Setup Contracts

**Purpose**: Define canonical and app repository structures for E2E test scenarios  
**Feature**: 003-e2e-integration-suite  
**Date**: November 22, 2025

---

## Canonical Repository

**Name**: `api-schemas`  
**Owner**: `testorg`  
**Type**: Canonical  
**Purpose**: Central API repository that receives published schemas from all app repositories

### Initialization

```bash
# Command executed in test
apx init canonical --org=testorg --repo=api-schemas --skip-git --non-interactive
```

### Expected Directory Structure

```
api-schemas/
├── proto/                    # Protocol Buffer schemas
│   └── .gitkeep
├── openapi/                  # OpenAPI specifications
│   └── .gitkeep
├── avro/                     # Avro schemas
│   └── .gitkeep
├── jsonschema/               # JSON Schema files
│   └── .gitkeep
├── parquet/                  # Parquet schemas
│   └── .gitkeep
├── buf.yaml                  # Buf configuration for org-wide lint/breaking policy
├── buf.work.yaml             # Buf workspace configuration
├── CODEOWNERS                # Per-path ownership rules
└── catalog/
    └── catalog.yaml          # API discovery catalog
```

### File Validations

| File | Validation Type | Expected Content |
|------|----------------|------------------|
| `buf.yaml` | File exists | - |
| `buf.yaml` | Contains pattern | `version: v2` |
| `buf.yaml` | Contains pattern | `use:\n  - DEFAULT` |
| `buf.work.yaml` | File exists | - |
| `buf.work.yaml` | Contains pattern | `version: v2` |
| `buf.work.yaml` | Contains pattern | `directories:\n  - proto` |
| `CODEOWNERS` | File exists | - |
| `CODEOWNERS` | Contains pattern | `# Default owner` |
| `catalog/catalog.yaml` | File exists | - |
| `catalog/catalog.yaml` | Contains pattern | `org: testorg` |
| `catalog/catalog.yaml` | Contains pattern | `repo: api-schemas` |
| `proto/.gitkeep` | File exists | - |
| `openapi/.gitkeep` | File exists | - |
| `avro/.gitkeep` | File exists | - |
| `jsonschema/.gitkeep` | File exists | - |
| `parquet/.gitkeep` | File exists | - |

### Git Setup

```bash
# Initialize git repository
git init
git config user.name "Test User"
git config user.email "test@example.com"

# Add remote pointing to Gitea
git remote add origin ${GITEA_URL}/testorg/api-schemas.git

# Initial commit
git add .
git commit -m "Initialize canonical repository"
git push -u origin main
```

---

## App Repository 1: Payment Service

**Name**: `payment-service`  
**Owner**: `testorg`  
**Type**: App (publisher)  
**Purpose**: Publishes payment ledger API to canonical repository

### Initialization

```bash
# Command executed in test
apx init app --org=testorg --non-interactive internal/apis/proto/payments/ledger/v1
```

### Expected Directory Structure

```
payment-service/
├── internal/
│   └── apis/
│       └── proto/
│           └── payments/
│               └── ledger/
│                   └── v1/
│                       ├── ledger.proto        # Payment ledger schema
│                       └── buf.yaml            # Module-specific buf config
├── apx.yaml                                    # APX configuration file
├── go.mod                                      # Go module file
└── go.work                                     # Go workspace (for overlays)
```

### File Validations

| File | Validation Type | Expected Content |
|------|----------------|------------------|
| `apx.yaml` | File exists | - |
| `apx.yaml` | Contains pattern | `kind: proto` |
| `apx.yaml` | Contains pattern | `module: payments.ledger.v1` |
| `apx.yaml` | Contains pattern | `org: testorg` |
| `apx.yaml` | Contains pattern | `version: v1` |
| `internal/apis/proto/payments/ledger/v1/ledger.proto` | File exists | - |
| `internal/apis/proto/payments/ledger/v1/ledger.proto` | Contains pattern | `syntax = "proto3";` |
| `internal/apis/proto/payments/ledger/v1/ledger.proto` | Contains pattern | `package payments.ledger.v1;` |
| `internal/apis/proto/payments/ledger/v1/buf.yaml` | File exists | - |

### Publishing Workflow

```bash
# Commit the schema
git add .
git commit -m "Add ledger API v1.0.0"

# Tag the module version (app repo format)
git tag proto/payments/ledger/v1/v1.0.0

# Publish to canonical repository
apx publish \
  --module-path=internal/apis/proto/payments/ledger/v1 \
  --canonical-repo=${GITEA_URL}/testorg/api-schemas.git \
  --version=v1.0.0
```

### Published Artifacts

**In canonical repository**:
- Branch: `publish/proto/payments/ledger/v1`
- Pull Request: Title "Publish proto/payments/ledger/v1"
- Tag (after PR merge): `proto/payments/ledger/v1.0.0`
- Files:
  ```
  proto/payments/ledger/v1/
  ├── ledger.proto
  └── buf.yaml
  ```

**Git History Validation**:
- All commits from app repo preserved in PR
- Commit authors/timestamps unchanged
- Commit messages intact

---

## App Repository 2: User Service

**Name**: `user-service`  
**Owner**: `testorg`  
**Type**: App (consumer + publisher)  
**Purpose**: Consumes payment API, publishes user profile API

### Initialization

```bash
# Command executed in test
apx init app --org=testorg --non-interactive internal/apis/proto/users/profile/v1
```

### Expected Directory Structure

```
user-service/
├── internal/
│   ├── apis/
│   │   └── proto/
│   │       └── users/
│   │           └── profile/
│   │               └── v1/
│   │                   ├── profile.proto       # User profile schema
│   │                   └── buf.yaml
│   └── gen/                                    # Generated code (git-ignored)
│       └── go/
│           └── proto/
│               └── payments/
│                   └── ledger@v1.0.0/          # Overlay for payment API
│                       └── ...
├── apx.yaml                                    # APX configuration
├── apx.lock                                    # Dependency lock file
├── go.mod                                      # Go module
└── go.work                                     # Go workspace with overlays
```

### Dependency Workflow

```bash
# Add dependency on payment API
apx add proto/payments/ledger/v1@v1.0.0

# Generate Go code (creates overlay)
apx gen go

# Sync go.work file
apx sync
```

### File Validations After Dependency Add

| File | Validation Type | Expected Content |
|------|----------------|------------------|
| `apx.lock` | File exists | - |
| `apx.lock` | Contains pattern | `proto/payments/ledger/v1` |
| `apx.lock` | Contains pattern | `ref: v1.0.0` |
| `apx.lock` | Contains pattern | `repo: .*testorg/api-schemas` |
| `internal/gen/go/proto/payments/ledger@v1.0.0/` | Directory exists | - |
| `go.work` | File exists | - |
| `go.work` | Contains pattern | `use ./internal/gen/go/proto/payments/ledger@v1.0.0` |

### Schema with Dependency

```protobuf
syntax = "proto3";

package users.profile.v1;

// Import from canonical path (resolves via go.work overlay)
import "github.com/testorg/apis/proto/payments/ledger/v1/ledger.proto";

message UserProfile {
  string user_id = 1;
  string email = 2;
  // Reference to payment ledger
  payments.ledger.v1.LedgerEntry default_payment_method = 3;
}
```

### Publishing Workflow

```bash
# Commit the schema (with dependency)
git add .
git commit -m "Add user profile API v1.0.0 with payment dependency"

# Tag the module version
git tag proto/users/profile/v1/v1.0.0

# Publish to canonical repository
apx publish \
  --module-path=internal/apis/proto/users/profile/v1 \
  --canonical-repo=${GITEA_URL}/testorg/api-schemas.git \
  --version=v1.0.0
```

### Published Artifacts

**In canonical repository**:
- Branch: `publish/proto/users/profile/v1`
- Pull Request: Title "Publish proto/users/profile/v1"
- Tag (after PR merge): `proto/users/profile/v1.0.0`
- Files:
  ```
  proto/users/profile/v1/
  ├── profile.proto
  └── buf.yaml
  ```

**Catalog Update** (after PR merge):
```yaml
# catalog/catalog.yaml
version: 1
org: testorg
repo: api-schemas
modules:
  - name: proto/payments/ledger/v1
    format: proto
    description: Payment ledger API
    version: v1.0.0
    path: proto/payments/ledger/v1
  - name: proto/users/profile/v1
    format: proto
    description: User profile API
    version: v1.0.0
    path: proto/users/profile/v1
```

---

## Edge Case Repositories

### App Repository 3: Conflicting Publisher

**Scenario**: Tests concurrent publication to same module path  
**Name**: `alternate-payment-service`  
**Module**: `proto/payments/ledger/v1` (conflicts with app1)  
**Expected Behavior**: Second publication should fail or create separate PR

### App Repository 4: Circular Dependency

**Scenario**: Tests circular dependency detection  
**Setup**:
1. App4 publishes `proto/service-a/v1`
2. App5 publishes `proto/service-b/v1` depending on service-a
3. App4 updates to depend on service-b (creates cycle)
**Expected Behavior**: Dependency add should fail with clear error

---

## Validation Contract Summary

### Canonical Repository Validations
- [x] Directory structure matches template
- [x] buf.yaml contains org-wide lint policy
- [x] buf.work.yaml includes proto directory
- [x] CODEOWNERS file exists
- [x] catalog.yaml has correct org/repo
- [x] All schema format directories have .gitkeep

### App Repository Validations
- [x] apx.yaml contains correct module metadata
- [x] Schema file exists at module path
- [x] buf.yaml exists for proto modules
- [x] go.mod and go.work exist for Go projects
- [x] Generated code in internal/gen/ (git-ignored)

### Publishing Validations
- [x] Git subtree split preserves commit history
- [x] PR created in canonical repo with correct title
- [x] PR contains all commits from app repo
- [x] Tags exist in both app and canonical repos
- [x] Tag formats differ (app: full path, canonical: relative)
- [x] Catalog updated after PR merge

### Dependency Validations
- [x] apx.lock created/updated with correct version
- [x] Overlay created in internal/gen/go
- [x] go.work updated with overlay path
- [x] Imports resolve to overlay during development
- [x] apx unlink removes overlay and updates go.work

---

## Test Data Fixtures

### Minimal Proto Schema (ledger.proto)

```protobuf
syntax = "proto3";

package payments.ledger.v1;

option go_package = "github.com/testorg/apis/proto/payments/ledger/v1;ledgerv1";

// Ledger entry for payment tracking
message LedgerEntry {
  string id = 1;
  string user_id = 2;
  int64 amount_cents = 3;
  string currency = 4;
  int64 timestamp = 5;
}

// Service for managing ledger entries
service LedgerService {
  rpc CreateEntry(CreateEntryRequest) returns (LedgerEntry);
  rpc GetEntry(GetEntryRequest) returns (LedgerEntry);
}

message CreateEntryRequest {
  LedgerEntry entry = 1;
}

message GetEntryRequest {
  string id = 1;
}
```

### Breaking Change Example (for edge case tests)

**Original** (v1.0.0):
```protobuf
message LedgerEntry {
  string id = 1;
  string user_id = 2;
  int64 amount_cents = 3;  // Required field
}
```

**Breaking** (attempted v1.1.0):
```protobuf
message LedgerEntry {
  string id = 1;
  // REMOVED: user_id (BREAKING CHANGE)
  int64 amount_cents = 3;
  string currency = 4;      // Added field (OK)
}
```

**Expected**: `apx breaking` detects removed field, blocks publication

---

## Next Steps

1. ✅ Contracts defined for canonical and app repositories
2. ✅ Validation criteria specified
3. ⏳ Generate quickstart.md (developer guide)
4. ⏳ Update agent context with E2E knowledge
