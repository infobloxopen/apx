# Frequently Asked Questions

Common questions about APX usage, configuration, and best practices.

## Architecture & Design

### Q: Do I need a `go.mod` in my app repo for authoring?
**A**: No. Buf ignores `go.mod` during schema authoring. `apx publish` will add a canonical `go.mod` in the PR to the canonical repo. If you do keep one locally, ensure it's canonical (no `replace` directives) and uses the correct module path, and it will be imported verbatim.

### Q: Will v2 code leak into v1 consumers?
**A**: No. v2 lives in a separate Go module (`.../v2` with its own `go.mod`). v1 imports never see v2 unless explicitly referenced. This follows Go's semantic import versioning.

### Q: Can APX discover APIs across the organization?
**A**: Yes. `apx search <keywords>` queries the catalog (generated in the canonical repo). You can also browse available tags and versions with `apx list apis`.

### Q: Can APX update dependencies automatically?
**A**: Yes. `apx update <api>` gets the latest patch/minor within the current major version. `apx upgrade <api>@<version>` targets a specific major version upgrade.

### Q: How does APX publish APIs?
**A**: APX uses git subtree to publish APIs, preserving complete commit history, authorship, and context in the canonical repository.

## Governance & Policy

### Q: How do we prevent service-specific options (e.g., gorm) in shared schemas?
**A**: `apx policy check` fails on banned annotations like `(gorm.*)` and unapproved generators. This runs in both app CI (pre-PR) and canonical CI to enforce organization policies.

### Q: Who can create tags in the canonical repo?
**A**: Only CI can create tags. Tag patterns like `proto/**/v*` are protected via GitHub branch protection rules. This ensures all releases go through validation and approval processes.

### Q: How do we handle breaking changes?
**A**: APX detects breaking changes automatically using format-specific tools (buf, oasdiff, etc.). Breaking changes require major version bumps. `apx version suggest` will recommend the appropriate version based on detected changes.

## Versioning & Compatibility

### Q: What's the difference between directory versions (v1/) and module versions?
**A**: 
- **Directory structure**: `proto/payments/ledger/v1/` contains schema files
- **Proto package**: `myorg.payments.ledger.v1` includes version suffix  
- **Go module path**: `github.com/org/apis/proto/payments/ledger` (v1 has no suffix)
- **Module path v2+**: `github.com/org/apis/proto/payments/ledger/v2` (includes suffix)

### Q: How do we handle schema evolution within a major version?
**A**: Within a major version (e.g., v1), you can make backwards-compatible changes:
- Add new fields (with appropriate defaults)
- Add new RPCs or methods
- Add new enum values (at the end)
- Make optional fields required is generally breaking

### Q: Can different teams use different versions of the same API?
**A**: Yes. Each major version is a separate Go module, so Team A can use `v1` while Team B uses `v2`. This allows gradual migration without forcing everyone to upgrade simultaneously.

## Development Workflow

### Q: Why are generated files not committed?
**A**: Generated code is deterministic based on `apx.lock`. Not committing it:
- Reduces merge conflicts
- Ensures consistency across environments  
- Forces regeneration in CI for verification
- Keeps repository clean and focused on source

### Q: How do we handle local development with dependencies?
**A**: 
1. Use `apx add` to pin dependencies in `apx.lock`
2. Run `apx gen <language>` to generate client code
3. Import generated code from `internal/gen/<language>/`
4. Never commit the `internal/gen/` directory

### Q: Can we author multiple schema formats in one app repo?
**A**: Yes. Configure multiple APIs in `apx.yaml`:
```yaml
apis:
  - kind: proto
    path: internal/apis/proto/payments/v1
    canonical: proto/payments/v1
  - kind: openapi  
    path: internal/apis/openapi/gateway/v1
    canonical: openapi/gateway/v1
```

## CI/CD & Automation

### Q: How do we integrate APX with existing CI pipelines?
**A**: Add APX validation to your existing workflows:
```yaml
- run: apx fetch --ci      # download tools
- run: apx lint           # validate schemas  
- run: apx breaking       # check compatibility
- run: apx policy check   # enforce governance
```

### Q: What happens if app CI fails after tagging?
**A**: The tag exists but no PR is created to the canonical repo. You can:
1. Fix the issues locally
2. Delete and recreate the tag
3. Push the corrected tag to trigger CI again

### Q: Can we customize the PR created by `apx publish`?
**A**: Yes, using flags:
```bash
apx publish \
  --pr-title="Custom title" \
  --pr-body="Custom description" \
  --pr-labels="api,breaking-change"
```

## Schema Formats

### Q: Does APX support formats other than Protocol Buffers?
**A**: Yes. APX supports:
- **Protocol Buffers** (primary, with buf integration)
- **OpenAPI** (with oasdiff and Spectral)
- **Avro** (with compatibility checking)
- **JSON Schema** (with diff analysis)
- **Parquet** (with custom compatibility rules)

### Q: Can we mix schema formats in the same API?
**A**: It's possible but not recommended. Each format has different versioning and compatibility rules. Keep formats separate for cleaner governance and tooling.

### Q: How does breaking change detection work for different formats?
**A**: Each format uses specialized tools:
- **Proto**: `buf breaking` (file and wire compatibility)
- **OpenAPI**: `oasdiff breaking` (API contract changes)  
- **Avro**: compatibility modes (BACKWARD, FORWARD, FULL)
- **JSON Schema**: schema diff analysis
- **Parquet**: additive nullable columns only

## Enterprise & Scale

### Q: How does APX handle large organizations with many teams?
**A**:
- **CODEOWNERS** provides per-API ownership
- **Domain organization** groups related APIs
- **Catalog search** helps discover existing APIs
- **Policy enforcement** ensures consistency
- **Gradual rollout** via feature flags

### Q: Can we use APX with private/internal dependencies?
**A**: Yes. APX works with private repositories by:
- Using appropriate Git credentials
- Configuring access tokens for CI
- Supporting VPN/proxy environments
- Allowing custom tool registries

### Q: How do we handle API deprecation?
**A**: 
1. Mark APIs as deprecated in schema comments
2. Update catalog metadata
3. Set sunset timelines in documentation
4. Use `apx list dependents` to find consumers
5. Coordinate migration with dependent teams

## Performance & Optimization

### Q: How does APX handle large repositories?
**A**: 
- **Git subtree** preserves complete commit history and authorship
- **Selective publishing** only processes changed APIs
- **Cache optimization** for tools and dependencies
- **Incremental processing** to handle repository scale efficiently

### Q: Can we run APX in containers for consistency?
**A**: Yes. Use `--use-container` flag or `APX_USE_CONTAINER=true` environment variable to run all tools in containers. This ensures consistent environments across development and CI.

### Q: How do we handle network restrictions?
**A**: APX respects standard proxy settings:
```bash
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080
apx fetch  # uses proxy for downloads
```

## Troubleshooting

### Q: What if `apx publish` fails with merge conflicts?
**A**: 
1. Ensure canonical repo is up to date
2. Resolve conflicts manually if needed
3. Contact API governance team for assistance
4. Consider rebasing your changes before publishing

### Q: How do we debug schema validation failures?
**A**: 
```bash
apx lint --verbose          # detailed error output
buf lint <specific-file>    # direct buf validation
apx config validate         # check configuration
```

### Q: What if dependencies can't be resolved?
**A**: 
1. Check canonical repo accessibility
2. Verify version exists: `apx show <api>`
3. Clear cache: `rm -rf ~/.cache/apx/`
4. Re-fetch tools: `apx fetch --force`

---

**Can't find your answer?** 
- Check the [Troubleshooting Guide](index.md)
- Open a [Discussion](https://github.com/infobloxopen/apx/discussions)
- Report an [Issue](https://github.com/infobloxopen/apx/issues)