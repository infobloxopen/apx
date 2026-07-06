# CRD Format

apx lifecycles a Kubernetes **CustomResourceDefinition** (CRD) as a first-class
schema format, alongside `proto`, `openapi`, `avro`, `jsonschema`, and
`parquet`. A CRD becomes a versioned, lint-checked, breaking-analyzed,
releasable, catalog-resolvable module — so a Kubernetes GVK (group / version /
kind) becomes a **versioned capability** that a dependency or an install-time
gate can reference and version-constrain.

apx **lifecycles** CRDs; it does not generate them. Authoring stays with
`controller-gen` / `kubebuilder`, exactly as apx does not itself emit OpenAPI or
proto code.

## Detection

apx recognizes a CRD by content: a YAML document whose `apiVersion` starts with
`apiextensions.k8s.io/` and whose `kind` is `CustomResourceDefinition`. Only
`apiextensions.k8s.io/v1` is supported; migrate `v1beta1` CRDs first. You can
also force the format with `--format crd`.

## Identity: GVK → module ID

A CRD's group/version/kind maps onto the module identity model:

| CRD field | Catalog slot | Example |
|---|---|---|
| `spec.group` | domain | `appkit.infoblox.dev` (already reverse-DNS) |
| `spec.names.kind` | name | `appcontract` |
| `spec.versions[].name` | API line | `v1alpha1` |

The module ID is `crd/<group>/<kind>/<version>`, for example
`crd/appkit.infoblox.dev/appcontract/v1alpha1`. There is **one apx module per
CRD version**: `v1alpha1` and `v1` of the same kind are distinct modules.

The Kubernetes version string is the API line. Its major (`v1alpha1` → 1,
`v2beta3` → 2) is the module's line major, so the module's semver major matches
it (`crd/.../v1alpha1` releases as `v1.x.x`). Maturity maps to lifecycle:

| CRD version | Lifecycle | Semver |
|---|---|---|
| `v1alpha1` | experimental | `v1.0.0-alpha.N` |
| `v1beta1` | beta | `v1.0.0-beta.N` |
| `v1` (GA) | stable | `v1.0.0` |

A CRD carries **no language bindings** — no Go module, npm package, Maven
artifact, etc. — because apx generates none for it.

## Lint

`apx lint <crd>` validates the Kubernetes structural-schema rules and CRD
conventions on top of the embedded `spec.versions[].schema.openAPIV3Schema`:

- `apiVersion` is `apiextensions.k8s.io/v1`; `metadata.name` is
  `<plural>.<group>`; `scope` is `Namespaced` or `Cluster`.
- Each version name is a valid Kubernetes version; names are unique; **exactly
  one** version sets `storage: true`; at least one is `served: true`.
- Every version has a `schema.openAPIV3Schema` whose root type is `object`.
- Every specified schema node has a valid type (`object`, `array`, `string`,
  `integer`, `number`, `boolean`) — unless it is an
  `x-kubernetes-preserve-unknown-fields` or `x-kubernetes-int-or-string` escape
  hatch. Arrays specify `items`; `properties` and a schema-form
  `additionalProperties` are not combined at the same level.

## Breaking analysis

`apx breaking <new> --against <old>` applies Kubernetes served-version
compatibility rules — which differ from raw OpenAPI diffing:

- Removing a served version, removing a field, narrowing a type, adding a
  required field, tightening a constraint (`maxLength`/`maxItems`/`maximum`
  down, `minLength`/`minItems`/`minimum` up, adding/changing `pattern`), adding
  or narrowing an `enum`, or dropping `x-kubernetes-preserve-unknown-fields` are
  all **breaking**.
- Additive changes (new optional field, new served version, relaxed constraint,
  added enum value, added `x-kubernetes-preserve-unknown-fields`) are **not**
  breaking.
- **Alpha versions carry no compatibility guarantee** (Kubernetes deprecation
  policy) — changes to a `vNalphaM` version are never flagged.

`apx semver` feeds the verdict into the version bump: a breaking change to a
served (beta/GA) version cannot bump within the line and must move to a new API
version — the Kubernetes rule expressed as apx semver.

## Release and catalog

CRD modules flow through the standard `apx release prepare` → `submit` →
`finalize` lifecycle and appear in `apx catalog generate` / `show` / `search` /
`inspect`, resolvable by their `crd/<group>/<kind>/<version>` ID. The catalog
entry carries the GVK and served/storage facts (`crd_group`, `crd_kind`,
`crd_scope`, `served_versions`, `storage_version`) so a consumer can
version-constrain the capability.

## Scope boundary

The catalog captures the **declared contract**: the schema, the GVK, served and
storage versions, and deprecation. It does not capture runtime behavior —
conversion-webhook logic, admission control, CEL (`x-kubernetes-validations`)
evaluation, and defaulting are out of scope.
