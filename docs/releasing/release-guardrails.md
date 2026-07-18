# Release Guardrails

APX enforces a set of guardrails during the release workflow to prevent invalid or contradictory releases.

## Lifecycle-Version Compatibility

APX validates that the declared lifecycle is compatible with the version being released:

| Lifecycle | Allowed versions | Rejected versions |
|-----------|-----------------|------------------|
| `experimental` | Must have `-alpha.*` prerelease | Stable versions, `-beta.*`, `-rc.*` |
| `beta` | Must have `-alpha.*`, `-beta.*`, or `-rc.*` prerelease | Stable versions |
| `stable` | Must be a stable version (no prerelease) | Any prerelease |
| `deprecated` | Any version | *(none)* |
| `sunset` | Releases are blocked | Any version (unless overridden) |

## v0 Line Restrictions

APIs on the `v0` line have additional restrictions:

| Guardrail | Rule |
|-----------|------|
| Lifecycle | Must be `experimental` or `beta` |
| Stable promotion | Not permitted — graduate to `v1` instead |
| Breaking changes | Allowed — minor version bump instead of rejection |

These restrictions are enforced by `apx release prepare` and `apx release promote`.

## Lifecycle Transition Rules

Lifecycle states must progress forward:

```
experimental → beta → stable → deprecated → sunset
```

APX rejects backward transitions (e.g., `stable` → `beta`).

## Breaking Change Enforcement

| Scenario | APX behavior |
|----------|-------------|
| Breaking change on `v0` line | Allowed — suggests minor version bump |
| Breaking change on `v1+` prerelease | Allowed with warning |
| Breaking change on `v1+` stable | Rejected — requires new major line |
| Breaking change on `deprecated` | Rejected — maintenance only |

## Client Build Verification

A spec can be valid OpenAPI 3 and pass `apx lint`/`apx breaking` yet still
produce a client that does not compile — for example redundant `_limit`/`limit`
query params that normalize to the same Go field, or a path parameter named
`url` that shadows the `net/url` import. `apx client verify` closes that gap: it
generates a client and **compiles** it, failing the release when any generated
client does not build.

| Scenario | APX behavior |
|----------|-------------|
| Generated client compiles | Passes |
| Generated client does not compile | Rejected — fails the gate |
| Generator toolchain absent (e.g. Node for TypeScript) | Skipped — not a failure |
| `--warn-only` (or `release.verify_clients.warn_only`) set | Failure downgraded to a warning (exit 0) |

Run it in release CI between `apx breaking` and `apx release prepare`. See
[`apx client verify`](../cli-reference/configuration.md#apx-client-verify--the-generate-and-compile-gate)
for configuration.

## See Also

- [Lifecycle Reference](lifecycle.md) — full lifecycle state definitions and policies
- [Versioning Strategy](../dependencies/versioning-strategy.md) — the three-layer versioning model
