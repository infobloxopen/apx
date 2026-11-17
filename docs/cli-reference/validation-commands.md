# Validation Commands

Commands for validating schemas, detecting breaking changes, classifying operations, and suggesting version bumps.

## `apx lint`

Validate schema files for syntax and style issues.

```bash
apx lint [path]
```

If `path` is omitted, APX detects schema directories from `module_roots` in `apx.yaml`. If `path` is an API ID (e.g. `proto/payments/ledger/v1`), APX resolves it to the local filesystem path.

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--format` | `-f` | string | auto-detected | Schema format: proto, openapi, avro, jsonschema, parquet |
| `--operation-ids` | | bool | `true` | OpenAPI only: fail when an operation has no `operationId` or an `operationId` repeats. Opt out with `--operation-ids=false`. |

### Format-Specific Validation

| Format | Tool | What it checks |
|--------|------|----------------|
| Protocol Buffers | `buf lint` | Naming conventions, package structure, field numbering, service definitions |
| OpenAPI | Native Go + Spectral | Built-in `operationId` check (see below), then schema structure, endpoint definitions, response formats |
| Avro | Native Go | Record structure, field type validity, required `name`/`fields` presence |
| JSON Schema | Native Go | JSON syntax, `$schema` URI, `type`, `properties`, `required` |
| Parquet | Native Go | Message-notation syntax, physical type validity, repetition levels |

For protobuf, APX also runs `go_package` validation — warning if the `go_package` option doesn't match the canonical import path.

### OpenAPI `operationId` check

For OpenAPI, `apx lint` has a built-in check that does not depend on Spectral or on a Spectral ruleset. It runs before Spectral, and it also runs when Spectral is not installed. It fails when:

- an operation has no `operationId`;
- an `operationId` repeats inside one document;
- an operation has an `x-operation-class` that is not `read`, `list`, `write` or `custom`.

The `operationId` is the stable name of an operation. Generated clients, dashboards and per-operation metrics use it as a key, so each operation must have one and each one must be unique.

```text
operationId check failed for ledger.yaml (2 problems):
  GET /entries/{id}: operationId is missing
  operationId "entries" is not unique: GET /entries, POST /entries
```

The check is on by default. A repository that cannot fix its specs immediately can turn it off with `apx lint --operation-ids=false`. `apx release finalize` runs the same check during re-validation and has the same flag.

### Examples

```bash
# Lint all schemas in the project
apx lint

# Lint a specific directory
apx lint internal/apis/proto/payments/ledger/v1

# Force format detection
apx lint --format proto internal/apis/

# Verbose output with details
apx lint --verbose
```

---

## `apx breaking`

Check for breaking changes between the current schemas and a baseline.

```bash
apx breaking [path] --against <ref>
```

The `--against` flag is required and specifies the baseline to compare against.

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--against` | | string | *(required)* | Git reference or path to compare against |
| `--format` | `-f` | string | auto-detected | Schema format |
| `--advisory` | | bool | `false` | Report breaking changes and exit 0 |
| `--operation-ids` | | bool | `true` | OpenAPI only: report a renamed or removed `operationId` as a breaking change. Opt out with `--operation-ids=false`. |

### Supported Baselines

```bash
# Compare against previous commit
apx breaking --against HEAD^

# Compare against a branch
apx breaking --against origin/main

# Compare against a specific tag
apx breaking --against proto/payments/ledger/v1/v1.0.0

# Compare against a specific commit
apx breaking --against abc1234
```

### Format-Specific Checks

| Format | Tool | Breaking changes detected |
|--------|------|---------------------------|
| Protocol Buffers | `buf breaking` | Field removal/renumbering, type changes, service/method removal |
| OpenAPI | `oasdiff breaking` + native Go | Endpoint removal, required field additions, response type changes, renamed or removed `operationId` |
| Avro | Native Go | New fields without defaults, type changes (BACKWARD/FORWARD/FULL/NONE modes) |
| JSON Schema | Native Go | Property removal, type change, required field additions |
| Parquet | Native Go | New required columns, removed columns, type/annotation changes, optional→required promotion |

### Examples

```bash
apx breaking --against HEAD^
apx breaking internal/apis/proto/ --against origin/main
apx breaking --format openapi --against v1.0.0
```

### OpenAPI `operationId` changes

`oasdiff breaking` rates a renamed or removed `operationId` as information (`api-operation-id-removed`), not as an error, so it does not fail on it. APX therefore compares the `operationId` of each operation of the baseline with the revision. Operations are matched by HTTP method and path; the names of path variables do not matter. A breaking finding is:

- an operation that stays on the same method and path with another `operationId`, or with none;
- an operation that is gone, when no operation of the revision has its `operationId`.

```text
operationId changes (1); a renamed or removed operationId breaks generated clients and per-operation history:
  GET /entries: operationId "listEntries" was renamed to "listAllEntries"
```

A finding fails the command like any other breaking change. `--advisory` reports it and exits 0. `--operation-ids=false` turns the comparison off. `apx release finalize` runs the same comparison against the previous release and has the same flag.

---

## `apx opclass`

Classify each operation of an OpenAPI 3.x or Swagger 2 document as `read`, `list`, `write` or `custom`.

```bash
apx opclass <spec>
```

The operation class is the value of the extension `x-operation-class`. It is separate from `x-aip-method`: `x-aip-method` marks only operations with the strict AIP request and response shapes, and client generators depend on that meaning. `apx opclass` never writes or changes `x-aip-method`.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `text` | Output format: `text` or `json`. The global `--json` flag also selects `json`. |
| `--write` | bool | `false` | Add `x-operation-class` to each operation of the file that lacks one |

### Rules

An `x-operation-class` that is already on an operation always wins. Its value must be `read`, `list`, `write` or `custom`. Without it, the first rule that matches decides. The `RULE` column of the output names the rule.

| Order | Rule | Match | Class |
|-------|------|-------|-------|
| 0 | `explicit` | `x-operation-class` is on the operation | its value |
| 1 | `aip` | `x-aip-method` is `Get` or `BatchGet` | `read` |
| | | `x-aip-method` is `List` or `Search` | `list` |
| | | `x-aip-method` is `Create`, `Update`, `Delete`, `Undelete`, `Purge`, `BatchCreate`, `BatchUpdate` or `BatchDelete` | `write` |
| 2 | `name` | The `operationId` starts or ends with the word `List` | `list` |
| | | The `operationId` starts or ends with the word `Read` or `Get` | `read` |
| | | The `operationId` starts or ends with the word `Create`, `Update`, `Patch` or `Delete` | `write` |
| 3 | `shape` | `GET` or `HEAD` on a path whose last segment is one variable (`/users/{id}`) | `read` |
| | | `GET` or `HEAD` on another path (`/users`) | `list` |
| | | `POST`, `PUT`, `PATCH` or `DELETE` | `write` |
| 4 | `default` | Nothing matched | `custom` |

`x-aip-method: Custom` and unknown `x-aip-method` values do not match rule 1. `GET` on the root path and `GET` on a path with a custom verb (`/users/{id}:export`) do not match rule 3.

**How rule 2 reads an `operationId`.** The id is split into words at each character that is not a letter or a digit, at each change from lower case to upper case (`getUser`), and at the end of an acronym (`HTTPServer`). The comparison ignores case. In the published protobuf form `Service_Method`, only the method part is used, and leading method words that repeat the service name are dropped: `Accounts_AccountListUsers` is read as `List`, `Users`. The first word decides when it is a verb; otherwise the last word decides. An id that starts with `Get` or `Read` and ends with `List` (`GetUserList`) is `list`.

Only whole words match: `Blacklist` and `Checklist` do not match `List`. The name rule cannot see a plural: `getDashboards` is `read`. When a rule gives the wrong class, set `x-operation-class` on the operation.

### Streaming

The `STREAMING` column is `true` when a response of the operation has the content type `text/event-stream` or `application/x-ndjson`, when a Swagger 2 `produces` list has one of them, or when the operation has `x-streaming: true`.

### Output

```text
METHOD  PATH          OPERATION ID                    CLASS  STREAMING  RULE
GET     /groups       AccountOrganizationGroups_List  list   false      name
POST    /groups       -                               write  false      shape
GET     /groups/{id}  AccountOrganizationGroups_Read  read   false      name

3 operations: read=1 list=1 write=1 custom=0
```

Operations are sorted by path, then by method. `PATH` is the path as the document writes it; the JSON output also has `basePath`, the path of `servers[0].url` (or the Swagger 2 `basePath`). A `-` in `OPERATION ID` means that the operation has no `operationId`.

```json
{
  "spec": "groups.yaml",
  "basePath": "/api/v1",
  "operations": [
    {"method": "GET", "path": "/groups", "operationId": "AccountOrganizationGroups_List", "class": "list", "streaming": false, "rule": "name"}
  ],
  "counts": {"custom": 0, "list": 1, "read": 0, "write": 0}
}
```

### Writing the class into the spec

`apx opclass <spec> --write` adds `x-operation-class` to each operation that lacks one. An operation that already has the extension keeps its value. With `--write`, the JSON output also has `added`, the number of operations that changed.

APX does not re-encode the file. It inserts text and leaves every other byte as it is: comments, key order, quoting, indentation and line endings.

- Block-style YAML: one new line after the `operationId` line, or before the first key of the operation when the `operationId` is absent or spans lines.
- Flow-style YAML and JSON: a quoted member, as the first member of the operation object.

Limits:

- An operation that the document reaches through a YAML alias (`get: *shared`) is not changed.
- An `x-operation-class` that arrives through a merge key (`<<`) is not seen; the operation gets its own.
- Only the first YAML document of a stream is read, and a `$ref` on a path item is not followed.

### Examples

```bash
# Print the table
apx opclass openapi/payments/ledger/v1/ledger.yaml

# JSON output
apx opclass ledger.yaml --format json

# Add x-operation-class where it is missing
apx opclass ledger.yaml --write
```

---

## `apx optable render`

Write the operation table of each API that a `RouteAPIBinding` names, and optionally the route table of the routes that the binding targets. Run it when you build a Helm chart: the chart then carries the tables to each cluster.

```bash
apx optable render --binding <file|chart> --out <dir> [flags]
```

### The binding

A `RouteAPIBinding` links the routes of one namespace to the APIs that they serve. Each entry pins one release of one API.

```yaml
apiVersion: traefik.infoblox.com/v1alpha1
kind: RouteAPIBinding
metadata: {name: ledger, namespace: payments}
spec:
  apis:
    - ref: openapi/payments/ledger/v1      # the API id
      version: v1.0.0                      # the pinned release
      externalPrefix: /api/v1              # optional; replaces the base path of the API contract
      targetRefs:
        - {group: networking.k8s.io, kind: Ingress, name: ledger}
```

The validation is strict:

- A document with another `apiVersion` is an error. Documents of other kinds in the input are skipped, so the rendered output of a chart is a valid input. An input with no `RouteAPIBinding` is an error.
- `ref`, `version` and at least one `targetRefs` entry are required. An unknown field in `spec` is an error.
- A `targetRefs` entry has `group: networking.k8s.io`, `kind: Ingress` and a `name`. It has no `namespace` field: the binding and its routes are in one namespace.
- One API can be in two entries, each with its own routes and its own `externalPrefix`. The two entries must pin the same version.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--binding` | string (repeatable) | required | A YAML file with one or more `RouteAPIBinding` documents, or a Helm chart directory. A chart directory is rendered with `helm template`; a file needs no `helm` binary. |
| `--spec` | string (repeatable) | | `<api-id>=<file>`: the OpenAPI 3.x or Swagger 2 file of one API, or the descriptor set of a `proto/` API (see [gRPC APIs](#grpc-apis)) |
| `--specs-dir` | string | | A directory with the layout of an API repository. The contract of an OpenAPI API is the single OpenAPI file in `<dir>/<api-id>/`. A `proto/` API needs `--spec`. |
| `--audience` | string (repeatable) | | `<api-id>=public` or `<api-id>=internal` |
| `--catalog` | string | | Path or URL of a catalog file that gives the audience of each API |
| `--out` | string | required | Output directory. The command creates it. |
| `--route-table` | bool | `false` | Also write `route-table.json` |
| `--route-table-configmap` | string | | Also write `route-table-configmap.yaml`, a ConfigMap with this name that holds the gzip route table. Implies `--route-table`. |
| `--namespace` | string | | `metadata.namespace` of the manifests, and the namespace of a binding that has none |
| `--format` | string | `text` | Output format of the summary: `text` or `json`. The global `--json` flag also selects `json`. |
| `--helm-set` | string (repeatable) | | `key=value` passed to `helm template --set` for a chart directory |
| `--release-name` | string | `optable` | Helm release name used when rendering a chart directory |

### Where the API contract comes from

`--spec <api-id>=<file>` wins. Otherwise the contract is the single OpenAPI file in `<specs-dir>/<api-id>/`; when that directory has more than one OpenAPI file, pass `--spec`. The command does not download a release: check out `--specs-dir` at the version that the binding pins. The `version` of the binding goes into the output as written.

Each operation must have an `operationId` (`apx lint` checks this).

A `proto/` API needs `--spec <api-id>=<file>` with its descriptor set. The command does not compile `.proto` files. Build the set at the pinned version with `buf build`, for example `buf build --path proto/payments/ledger/v1 -o ledger.binpb`.

### Audience

The audience of an API is `public` or `internal`. The first source that has a value decides:

1. `--audience <api-id>=public|internal`
2. The `audience` of the catalog entry, from `--catalog`, or from `catalog/catalog.yaml` in `--specs-dir` when that file is there
3. `internal`

### External path

The `path` of an operation in the output is the external path template: the path that the client sends. It is the `externalPrefix` of the binding entry when it is set, otherwise the base path of the contract (the path of `servers[0].url`, or the Swagger 2 `basePath`), joined with the path of the operation. The result has single slashes and no trailing slash, and `{name}` segments stay as written. Set `externalPrefix: /` when the client path has no prefix.

| `externalPrefix` | Base path | Operation path | External path |
|---|---|---|---|
| | `/v2` | `/accounts/{id}` | `/v2/accounts/{id}` |
| `/api/v2/` | `/internal/v2` | `/accounts/` | `/api/v2/accounts` |
| `/` | `/v2` | `/accounts` | `/accounts` |

### gRPC APIs

The contract of a `proto/` API is a descriptor set: the output of `buf build -o <file>`. A file that ends in `.json` is the JSON form (`buf build -o <file>.json`); any other file is the binary form. A plain `google.protobuf.FileDescriptorSet` (`protoc --include_imports --descriptor_set_out`) is also accepted.

Each method of each service is one operation:

| Field | Value |
|---|---|
| `method` | `POST` |
| `path` | `/<package>.<Service>/<Method>`, for example `/payments.ledger.v1.Ledger/GetEntry` |
| `operationId` | `<Service>_<Method>`, for example `Ledger_GetEntry`: the form that a gRPC gateway gives to the same method in OpenAPI |
| `class` | From the `operationId`, with the name rule of [`apx opclass`](#apx-opclass). A method that the name rule does not decide is `write`, because a gRPC call is a `POST`. |
| `streaming` | `true` when the client or the server streams |

The rules:

- A Buf image marks the files that the API imports. Only the files that are not imports give operations. A plain descriptor set has no marks, so each service in it gives operations.
- The set must be complete: each import of each file must be in the set. A reader of the set, such as a web application firewall, then can decode each request and response message of a call. A set that is not complete is an error.
- A set with no service is an error.
- A gRPC client always sends `/<package>.<Service>/<Method>`, so the path has no prefix. An `externalPrefix` other than `/` on a `proto/` API is an error.

### Precedence between APIs

More than one API can be bound to one route. For each route, and for each method and path, the command selects one API:

1. A literal segment wins over a `{name}` segment. `/accounts/groups` and `/accounts/{id}` are both kept; the reader of the route table prefers the literal segment, segment by segment from the left.
2. A `public` API wins over an `internal` API, when the two operations have the same `operationId`.
3. Any other overlap is a conflict: two APIs with the same audience on one method and path, or one method and path with different `operationId` values. Two templates that differ only in a variable name (`/accounts/{id}`, `/accounts/{account_id}`) are the same path.

A conflict makes the command fail. The error names the route, the method, the path and the two APIs, and the command writes no file:

```text
Error: 1 operation conflict(s) between the APIs of the binding:
  route identity/identity-api: GET /v2/accounts: API openapi/iam/identity-internal/v2 and API openapi/iam/identity/v2 are in conflict: the two APIs have the same audience (internal)
```

APIs that share no route are not compared.

### Output files

**One ConfigMap manifest for each API**, in the file `optable-<hash>.yaml`. The name is `optable-` and the first 10 hex digits of the SHA-256 of the API id.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: optable-9e288350b7
  labels:
    apx.infoblox.com/operation-table: "true"
  annotations:
    apx.infoblox.com/api: openapi/reports/server/v1
    apx.infoblox.com/api-version: v1.0.0
data:
  operations.json: |
    {
      "schema": "apx.infoblox.com/operation-table/v1",
      "api": "openapi/reports/server/v1",
      "version": "v1.0.0",
      "audience": "internal",
      "operations": [
        {
          "method": "GET",
          "path": "/api/v1/dashboards/{id}",
          "operationId": "getDashboard",
          "class": "read",
          "streaming": false
        }
      ]
    }
```

`operations` is sorted by `path`, then by `method`. `class` and `streaming` come from the classifier of [`apx opclass`](#apx-opclass). When one API has two `externalPrefix` values, its table has the operations under each prefix.

The table of a `proto/` API also has `descriptor`: the ConfigMap that holds the descriptor set, its `binaryData` key and the SHA-256 digest of the set.

```json
"descriptor": {
  "configMap": "optable-4def501dd8-descriptors",
  "key": "descriptors.binpb.gz",
  "digest": "sha256:<64 hex digits>"
}
```

**One descriptor ConfigMap for each `proto/` API**, in the file `optable-<hash>-descriptors.yaml`, with the name of the operation-table ConfigMap and `-descriptors`. It has the label `apx.infoblox.com/descriptor-set: "true"`, the annotations `apx.infoblox.com/api`, `apx.infoblox.com/api-version` and `apx.infoblox.com/descriptor-digest`, and the `binaryData` key `descriptors.binpb.gz`. The key holds the gzip bytes of the canonical set: a binary `google.protobuf.FileDescriptorSet` with the files sorted by name, without source code info and without the Buf marks. The digest is the SHA-256 of these bytes before gzip. A reader finds the set of a route operation by its `api`, and checks the digest.

**`route-table.json`** (with `--route-table`): each route with its resolved operations. Each method and path is there one time for each route.

```json
{
  "schema": "traefik.infoblox.com/route-table/v1",
  "routes": [
    {
      "namespace": "reports",
      "name": "reports",
      "defaultApi": "openapi/reports/server/v1",
      "operations": [
        {
          "method": "GET",
          "path": "/api/v1/dashboards/{id}",
          "api": "openapi/reports/server/v1",
          "operation": "getDashboard",
          "streaming": false
        }
      ]
    }
  ]
}
```

A route is one Ingress. Its namespace is the `metadata.namespace` of the binding, or `--namespace` when the binding has none; the route table needs one of them. `defaultApi` is the API with the highest precedence on the route: the first `public` API in binding order, otherwise the first API. Routes are sorted by namespace, then by name.

**`route-table-configmap.yaml`** (with `--route-table-configmap <name>`): a ConfigMap whose `binaryData` key `route-table.json.gz` holds the gzip bytes of `route-table.json`.

The output is deterministic: the same inputs give the same bytes.

### Summary

The text summary has one line for each API, one line for each route and the list of files. `--format json` prints:

```json
{
  "apis": [
    {
      "api": "openapi/reports/server/v1",
      "version": "v1.0.0",
      "audience": "internal",
      "spec": "reports.yaml",
      "configMap": "optable-9e288350b7",
      "descriptor": "",
      "operations": 4,
      "classes": {"custom": 1, "list": 1, "read": 1, "write": 1}
    }
  ],
  "routes": [
    {"namespace": "reports", "name": "reports", "defaultApi": "openapi/reports/server/v1", "operations": 4}
  ],
  "conflicts": [],
  "files": ["optable-9e288350b7.yaml"]
}
```

`descriptor` is the name of the descriptor ConfigMap of a `proto/` API, and empty for an OpenAPI API. On a conflict, `conflicts` has one entry for each conflict (`namespace`, `route`, `method`, `path`, `api`, `otherApi`, `reason`), `files` is empty and the exit code is not zero.

### Limits

- OpenAPI files and gRPC descriptor sets only. The command does not compile `.proto` files.
- The gzip descriptor set of one API must be at most 700 KiB, because a ConfigMap holds at most 1 MiB.
- The class of a gRPC method comes from its name only; no method option sets it.
- The command does not fetch a pinned release from the catalog; see "Where the API contract comes from".
- The route kind is `Ingress` only.

### Examples

```bash
# One API, the contract from a local file
apx optable render --binding binding.yaml \
    --spec openapi/payments/ledger/v1=ledger.yaml --out chart/files

# A chart, the contracts from a checkout of the API repository, with the route table
apx optable render --binding helm/payments --specs-dir ../apis \
    --out rendered --route-table --namespace payments

# A gRPC API: build its descriptor set, then render
buf build --path proto/payments/ledger/v1 -o ledger.binpb
apx optable render --binding binding.yaml \
    --spec proto/payments/ledger/v1=ledger.binpb --out chart/files --route-table

# The route table as a gzip ConfigMap, and a JSON summary
apx optable render --binding binding.yaml --specs-dir ../apis \
    --out rendered --route-table-configmap route-table --format json
```

---

## `apx pathlint`

Compare the HTTP paths that the Ingress objects of a chart serve with the paths that the OpenAPI specs declare.

```bash
apx pathlint --ingress <chart|manifest> --spec <file> [flags]
```

The report has three parts: ingress paths that no spec declares, spec paths that no ingress rule routes, and the matched pairs. The command exits non-zero when one of the first two parts is not empty. With `--warn-only` it always exits 0 and the report is the same.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--ingress` | string (repeatable) | required | A chart directory (rendered with `helm template`) or a rendered manifest YAML file |
| `--spec` | string (repeatable) | required | An OpenAPI 3.x or Swagger 2 file. With `--binding`, also `<api-id>=<file>`. |
| `--external-prefix` | string | | An external path prefix that replaces the base path of every spec |
| `--binding` | string | | A `RouteAPIBinding` YAML file or a chart directory. The `externalPrefix` of the matching API replaces the base path of its spec. |
| `--format` | string | `text` | Output format of the report: `text` or `json` |
| `--helm-set` | string (repeatable) | | `key=value` passed to `helm template --set` for a chart directory |
| `--release-name` | string | `pathlint` | Helm release name used when rendering a chart directory |
| `--warn-only` | bool | `false` | Print the report and always exit 0 |
| `--out` | string | | Write the report to this file and not to stdout |

### External prefix

The external path of a spec path is the base path of the spec (the path of `servers[0].url`, or the Swagger 2 `basePath`) plus the path. Without `--external-prefix` and `--binding`, that base path applies.

Use a prefix when the client path does not start with the base path of the spec, for example behind a rewrite rule:

- `--external-prefix /api/v2` replaces the base path of every `--spec`.
- `--binding <file|chart>` takes the `externalPrefix` of each API from a `RouteAPIBinding` (see [`apx optable render`](#apx-optable-render)). Write a spec as `<api-id>=<file>` to name its API. A plain file is accepted when the binding has one API. An API without `externalPrefix` keeps the base path of its spec. An API with two `externalPrefix` values is checked under each of them. An API that has one entry with `externalPrefix` and one entry without it is an error in `pathlint`.

The two flags cannot be used together.

### JSON report

`--format json` prints `ingressInputs`, `specInputs`, `ingressRules`, `specPaths`, the lists `undeclared` (`path`, `kind`, `host`, `source`), `unreachable` (`path`, `spec`, `base`) and `matched` (`path`, `example`), `matchedPairs`, and `result`: `PASS`, `WARN` (drift with `--warn-only`) or `FAIL`.

### Examples

```bash
apx pathlint --ingress rendered.yaml --spec identity.swagger.json
apx pathlint --ingress helm/identity --spec v1.json --spec v2.json --warn-only
apx pathlint --ingress rendered.yaml --spec spec.yaml --external-prefix /api/v2
apx pathlint --ingress helm/identity --binding helm/identity \
    --spec openapi/iam/identity/v2=identity.yaml --format json
```

---

## `apx semver`

Suggest a semantic version bump based on schema changes.

```bash
apx semver [path] --against <ref>
```

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--against` | | string | *(required)* | Git reference to compare against |
| `--api-id` | | string | `""` | API ID (e.g. proto/payments/ledger/v1) |
| `--lifecycle` | | string | `""` | Lifecycle state |
| `--format` | `-f` | string | auto-detected | Schema format |

### How It Works

1. Runs `apx breaking` to detect breaking changes
2. Lists existing git tags to find the current latest version
3. Applies versioning rules:

| Change type | Bump | Rationale |
|-------------|------|-----------|
| Breaking changes | **Rejected** | Requires a new major API line (e.g. v1 → v2) |
| Non-breaking additive changes | **Minor** | New fields, RPCs, or endpoints |
| No schema changes | **Patch** | Documentation, metadata, or tooling-only changes |

4. Applies lifecycle mapping for prerelease tags:

| Lifecycle | Prerelease format |
|-----------|------------------|
| `experimental` | `-alpha.<n>` |
| `beta` | `-beta.<n>` |
| `stable` | *(none)* |

### Examples

```bash
# Suggest based on changes since last commit
apx semver --against HEAD^
# → minor (new fields added, no breaking changes)

# With explicit API ID and lifecycle
apx semver --api-id proto/payments/ledger/v1 --lifecycle beta --against HEAD^
# → v1.1.0-beta.1

# JSON output
apx --json semver --against origin/main
```

---

## `apx policy check`

Check schema files against organization policies.

```bash
apx policy check [path]
```

Validates schemas against policy rules configured in `apx.yaml`:

- **Forbidden proto options** — rejects schemas using banned annotations (e.g. `gorm.*`)
- **Allowed proto plugins** — ensures only approved code generation plugins are used
- **OpenAPI ruleset** — applies custom Spectral rulesets
- **Avro compatibility** — enforces compatibility mode (BACKWARD, FORWARD, FULL)
- **Parquet rules** — enforces additive nullable-only column additions

### Example

```bash
apx policy check
apx policy check internal/apis/proto/
```

---

## Validation in CI

A typical CI pipeline runs all three validation commands:

```yaml
- name: Validate schemas
  run: |
    apx lint
    apx breaking --against origin/main
    apx policy check
```

## See Also

- [Release Guardrails](../releasing/release-guardrails.md) — version and lifecycle enforcement
- [Versioning Strategy](../dependencies/versioning-strategy.md) — the three-layer versioning model
- [Buf Issues](../troubleshooting/buf-issues.md) — troubleshooting Buf-related problems
