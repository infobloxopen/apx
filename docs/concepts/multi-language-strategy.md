# Multi-Language Strategy

APX provides a symmetric developer experience across languages. The core principle: **schemas are the published artifact, not generated stubs**. Code is always generated locally using each language's native toolchain and packaging conventions.

## Design Principles

1. **Schema-first publishing** -- APX releases schema files (`.proto`, OpenAPI specs, Avro schemas) to a canonical repository or artifact registry. Generated code is never published.

2. **Local generation** -- Every consumer generates code locally. This gives teams control over toolchain versions (protoc, grpc, etc.) and avoids version coupling between producer and consumer.

3. **Same identity everywhere** -- The same API identity (`proto/payments/ledger/v1`) deterministically derives coordinates for every language. Dev and prod use the same identity; only the resolution backend changes.

4. **Native packaging** -- Each language uses its own ecosystem's conventions. Go developers see Go modules, Python developers see pip packages, Java developers see Maven coordinates.

## Language Support Matrix

| Concern | Go | Python | Java | TypeScript |
|---------|-----|--------|------|------------|
| Published artifact | Go module | sdist / wheel | Schema zip / jar | npm package |
| Local overlay | `go.work use` | `pip install -e` | `mvn install:install-file` | `npm link` |
| Resolution mechanism | go.work -> go.mod | pkgutil namespace | Maven dependency resolution | npm / workspace |
| Code generation | `apx gen go` | `apx gen python` | `mvn generate-sources` | `apx gen typescript` (planned) |
| Dev command | `apx sync` | `apx link python` | `apx link java` (planned) | `apx link typescript` (planned) |
| Unlink hint | `go get ...` | `pip install ...` | pom.xml dependency | `npm install ...` |
| Status | **Tier 1** | **Tier 2** | **Tier 2** | **Tier 2** |

### Tier definitions

- **Tier 1** -- Full overlay lifecycle: generate, sync, link, unlink, release. First-class CI support.
- **Tier 2** -- Identity derivation, generation, and local development. Release and CI support in progress.
- **Planned** -- Identity derivation designed but not yet implemented.

## Identity Derivation by Language

Given `org=acme` and API path `proto/payments/ledger/v1`:

| Language | Coordinate Type | Derived Value |
|----------|----------------|---------------|
| Go | Module | `github.com/acme/apis/proto/payments/ledger` |
| Go | Import | `github.com/acme/apis/proto/payments/ledger/v1` |
| Python | Dist name | `acme-payments-ledger-v1` |
| Python | Import | `acme_apis.payments.ledger.v1` |
| Java | Maven coords | `com.acme.apis:payments-ledger-v1-proto` |
| Java | Package | `com.acme.apis.payments.ledger.v1` |
| TypeScript | npm package | `@acme/payments-ledger-v1-proto` |

## Go Workflow

Go is the Tier 1 language with the most mature overlay system:

1. `apx gen go` generates code into `internal/gen/go/` with a synthesized `go.mod` declaring the canonical module path.
2. `apx sync` adds `use` directives to `go.work` so the Go toolchain resolves canonical imports to local overlays.
3. When ready for production, `apx unlink` removes the overlay and `go get` adds the released module. Import paths stay the same.

## Python Workflow

Python uses editable installs (`pip install -e`) as the local resolution mechanism:

1. `apx gen python` scaffolds each overlay as an installable Python package with `pyproject.toml` and namespace `__init__.py` files.
2. `apx link python` runs `pip install -e` for each overlay into the active virtualenv.
3. `apx unlink` removes the overlay; `pip install <dist-name>` adds the released package. Import paths stay the same.

## Java Workflow (Maven-Native)

Java uses Maven's dependency resolution and code generation phases:

1. **Producer** releases schema artifacts (proto files packaged as a jar/zip) to a Maven repository via APX's release pipeline.
2. **Consumer** adds the schema artifact as a Maven dependency using the derived coordinates (`com.<org>.apis:<domain>-<name>-<line>-proto`).
3. Maven's `generate-sources` phase uses `protobuf-maven-plugin` (or equivalent) to generate Java code from the schema artifact into `target/generated-sources/`.
4. For local development, `apx link java` (planned) installs schema artifacts to `~/.m2/repository`, allowing Maven to resolve them without a remote repository.

Java developers never interact with Go modules or `go.work`. The Maven coordinate system provides a complete, self-contained experience.

## TypeScript Workflow

TypeScript uses npm packages as the published artifact with scoped package names:

1. **Producer** releases schema artifacts to an npm registry via APX's release pipeline. The npm package name is deterministically derived: `@<org>/<domain>-<name>-<line>-proto`.
2. **Consumer** installs the package using `npm install @<org>/<domain>-<name>-<line>-proto`.
3. For local development, `apx link typescript` (planned) links local schema artifacts via `npm link`, allowing resolution without a remote registry.
4. `apx unlink` removes the local link; `npm install @<org>/<pkg>` adds the released package. Import paths stay the same.

TypeScript developers import generated types directly from the npm package name:

```typescript
import { LedgerService } from "@acme/payments-ledger-v1-proto";
```
