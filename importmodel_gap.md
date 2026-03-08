# Import Model — Gap Analysis

This document captures what was **not** implemented from `importmodel.md` and why.

Ref: `importmodel.md` sections A–G, section 2 (developer scenarios).

---

## Implemented (for reference)

| Section | Requirement | Status |
|---------|------------|--------|
| A | Canonical identity model (api, source, releases, languages) | Done — `config.go` types, `schema.go` field tree |
| B | One repo as default truth | Done — no `apis-go` references; docs use `github.com/<org>/apis` |
| C | API line vs release versioning | Done — `APIIdentity.Line` + `ReleaseInfo.Current` + `DeriveTag()` |
| D | Lifecycle metadata first-class | Done — `ValidateLifecycle()`, schema enum, config field |
| E | Path derivation rules (v1 no suffix, v2+ /vN) | Done — `DeriveGoModule()`, `DeriveGoImport()`, 12 unit tests |
| F.1 | Read the API ID | Done — `ParseAPIID()` in `identity.go` |
| F.2 | Derive canonical source path | Done — `DeriveSourcePath()` |
| F.3 | Derive language coordinates | Done — `DeriveGoModule()`, `DeriveGoImport()`, `DeriveLanguageCoords()` |
| F.6 | Create subdirectory tag | Done — `DeriveTag()` + `publishWithIdentity()` |
| G | Identity inspection commands | Done — `inspect identity`, `inspect release`, `explain go-path` |

---

## Gap 1: Validate `go_package` against derived identity (F.4)

**importmodel.md says:**

> validate `go_package` and module path

When `apx publish` runs, it should read the `.proto` files under the API's source path, extract `option go_package`, and verify it matches the derived import path (`DeriveGoImport()`).

**Why it was not done:**

1. **Proto file parsing is non-trivial.** The proto file parser needs to handle `option go_package` with both simple (`"path"`) and compound (`"path;alias"`) syntax. APX does not currently have a proto AST parser — it delegates to `buf` for lint/breaking. Adding one just for `go_package` extraction introduces a new dependency or a fragile regex-based parser.

2. **Multi-file ambiguity.** A module path like `proto/payments/ledger/v1` may contain multiple `.proto` files. Each may declare its own `go_package`. The validation logic must decide: validate all of them? Only the first? Fail on any mismatch? Those rules need a design decision.

3. **Non-proto formats have no equivalent.** OpenAPI, Avro, JSON Schema, and Parquet don't have a `go_package` concept. The validation would only apply to proto, making it format-specific logic inside a format-agnostic publish path.

4. **buf already validates `go_package`.** When `apx lint` runs (via buf), it can enforce `go_package` rules through buf's `PACKAGE_DIRECTORY_MATCH` and managed mode. Duplicating that in publish adds a second enforcement point.

---

## Gap 2: Create or validate canonical `go.mod` (F.5)

**importmodel.md says:**

> create or validate the canonical `go.mod`

During publish, APX should generate a `go.mod` file in the canonical repo's subdirectory (e.g., `proto/payments/ledger/go.mod` for v1, `proto/payments/ledger/v2/go.mod` for v2+), or validate an existing one matches the derived module path.

**Why it was not done:**

1. **`go.mod` generation requires Go module ecosystem knowledge.** The generated `go.mod` needs a correct `module` directive, a valid `go` directive (which Go version?), and potentially `require` entries for generated code dependencies (`google.golang.org/protobuf`, `google.golang.org/grpc`). Getting the dependency list right requires either running `go mod tidy` (which needs generated code to exist) or maintaining a hardcoded dependency list (which becomes stale).

2. **Publish currently uses git subtree.** The `SubtreePublisher` pushes a subdirectory as a subtree split. The `go.mod` must exist in the subtree *before* the split, meaning it must be committed to the source repo first. This creates a chicken-and-egg: publish needs `go.mod`, but `go.mod` needs to be in the repo before publish runs.

3. **Canonical CI is the documented owner.** APX's own docs say canonical CI is responsible for official releases. The `go.mod` could be created by CI (e.g., a GitHub Action that runs `go mod init` + `go mod tidy` after code generation), which is the more reliable integration point.

4. **v1 vs v2+ placement is tricky.** For v1, `go.mod` lives at `proto/payments/ledger/go.mod`. For v2+, it lives at `proto/payments/ledger/v2/go.mod`. This means the directory where `go.mod` lives differs from the API source path for v1 (source is at `.../v1/`, module root is at `.../`). This requires additional logic to compute the correct `go.mod` location.

---

## Gap 3: Record lifecycle/version in catalog on publish (F.7)

**importmodel.md says:**

> record lifecycle/version in catalog

When publish completes, APX should update `catalog/catalog.yaml` in the canonical repo to reflect:

```yaml
apis:
  - id: proto/payments/ledger/v1
    latest_release: v1.0.0-beta.1
    lifecycle: beta
    owners:
      - team-payments
```

**Why it was not done:**

1. **Catalog schema doesn't support identity fields yet.** The current `catalog.Module` struct has `Name`, `Format`, `Description`, `Version`, `Path`, `Tags`, `Owners`. It does not have `ID` (API identity format), `Lifecycle`, `LatestRelease`, or `LatestPrerelease`. Adding these fields changes the catalog schema, which affects `apx search`, catalog generation during `apx init canonical`, and any CI that reads catalog files.

2. **Publish runs in the app repo, catalog lives in the canonical repo.** The publisher does a subtree split/push to the canonical repo. Updating `catalog.yaml` requires either: (a) cloning the canonical repo, editing the file, committing, and pushing — a separate git workflow; or (b) doing it as part of the subtree push, which means the catalog update must be committed alongside the module content.

3. **Concurrency / merge conflicts.** If two teams publish different APIs at the same time, both will try to update `catalog.yaml`. This needs either an atomic update mechanism (like a separate commit + rebase loop) or a CI-based approach where catalog is regenerated from git tags rather than updated incrementally.

4. **Catalog regeneration is the safer model.** Instead of updating catalog on each publish, APX could regenerate it from tags/filesystem during CI (which it already does with `Generator.Scan()`). This avoids merge conflicts and ensures catalog is always consistent with what's actually in the repo.

---

## Gap 4: `apx show` command (Section 2 — search/discovery)

**importmodel.md says:**

```
apx search ledger
apx show proto/payments/ledger/v1
```

With output including lifecycle, latest stable, latest prerelease, Go module/import, owners.

**Why it was not done:**

1. **`apx inspect identity` already covers most of this.** Running `apx inspect identity proto/payments/ledger/v1 --source-repo github.com/acme/apis` prints API ID, format, domain, name, line, lifecycle, source path, Go module, Go import. The output is nearly identical to what `apx show` would produce.

2. **"Latest stable" and "latest prerelease" require querying git tags.** `apx show` would need to list all tags matching `proto/payments/ledger/v1/v*`, parse them as semver, separate prereleases from stable releases, and find the latest of each. This requires either access to the canonical repo's git history or a catalog that tracks releases (see Gap 3).

3. **`apx show` is a catalog-dependent command.** Without Gap 3 (catalog recording), `apx show` can only derive identity fields (which `inspect identity` already does). The additional value — latest versions, owners, lifecycle from the catalog — requires the catalog to have those fields first.

4. **Naming: `show` vs `inspect`.** Introducing `apx show` alongside `apx inspect identity` creates UX ambiguity. A cleaner approach might be to extend `apx inspect identity` to pull catalog data when available, rather than adding a separate command.

---

## Gap 5: `apx lint` / `apx breaking` with API ID argument (Section 2)

**importmodel.md shows:**

```bash
apx lint proto/payments/ledger/v1
apx breaking proto/payments/ledger/v1
```

Currently `apx lint` and `apx breaking` take a file path, not an API ID.

**Why it was not done:**

1. **`lint` and `breaking` delegate to buf.** They pass the path argument directly to `buf lint` / `buf breaking`. Buf expects a file or directory path, not an API ID.

2. **API ID → path resolution requires config context.** To resolve `proto/payments/ledger/v1` to a filesystem path, APX needs to know the module root (from `apx.yaml` `module_roots`) and the working directory. This resolution exists in other commands but hasn't been wired into lint/breaking.

3. **This is a UX improvement, not a model gap.** The identity model is fully implemented. This is about making `lint` and `breaking` understand API IDs as a convenience, which is a separate feature.

---

## Summary

| Gap | importmodel ref | Blocking? | Depends on |
|-----|----------------|-----------|------------|
| 1. `go_package` validation | F.4 | No — buf already validates | Proto parser or buf integration |
| 2. `go.mod` creation | F.5 | No — can be CI-driven | Go module ecosystem, subtree workflow |
| 3. Catalog recording | F.7 | No — catalog can be regenerated from tags | Catalog schema update, concurrency model |
| 4. `apx show` command | Section 2 | No — `inspect identity` covers core | Gap 3 (catalog with releases/lifecycle) |
| 5. lint/breaking with API ID | Section 2 | No — works with paths today | API ID → path resolution |

None of these gaps block the identity model from being usable. They are all enhancements that build on the foundation.
