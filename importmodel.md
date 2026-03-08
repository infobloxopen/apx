

## Target model

APX should adopt **one canonical source repo** and one **repo-neutral API identity**, then derive language-specific package/module names from that identity.

The cleanest model is:

* **Canonical source repo:** `github.com/<org>/apis`
* **Canonical API ID:** `<format>/<domain>/<name>/<api-line>`

  * example: `proto/payments/ledger/v1`
* **Release version:** SemVer tag on that API line

  * examples: `v1.0.0-alpha.1`, `v1.0.0-beta.1`, `v1.0.0`
* **Lifecycle metadata:** separate field in catalog/config

  * `experimental`, `beta`, `stable`, `deprecated`, `sunset`

That keeps three things separate:

1. **API identity**: what contract you are talking about
2. **artifact version**: which published build you want
3. **lifecycle state**: how much confidence/support it has

That is the simplest way to support versioning, alpha/beta/stable releases, and clean imports without making paths explode.

It also fits APX’s current overall direction better than the current mixed story. APX docs say the canonical repo is `github.com/<org>/apis`, call it the single source of truth, and show a canonical repo layout where v1 and v2 are represented as versioned API lines inside that repo. At the same time, the quickstart also introduces `apis-go` examples, which is where the current identity confusion comes from. ([Infoblox Open][1])

---

## What this would look like

For a new API:

* **API ID:** `proto/payments/ledger/v1`
* **proto package:** `acme.payments.ledger.v1`
* **source repo path:** `github.com/acme/apis/proto/payments/ledger/v1`
* **Go package import path:** `github.com/acme/apis/proto/payments/ledger/v1`
* **Go module path for v1 line:** `github.com/acme/apis/proto/payments/ledger`
* **published versions for that API line:**

  * `proto/payments/ledger/v1/v1.0.0-alpha.1`
  * `proto/payments/ledger/v1/v1.0.0-beta.1`
  * `proto/payments/ledger/v1/v1.0.0`

For a breaking change:

* **new API ID:** `proto/payments/ledger/v2`
* **Go package import path:** `github.com/acme/apis/proto/payments/ledger/v2`
* **Go module path:** `github.com/acme/apis/proto/payments/ledger/v2`
* **published versions:**

  * `proto/payments/ledger/v2/v2.0.0-alpha.1`
  * `proto/payments/ledger/v2/v2.0.0`

This matches APX’s documented canonical repo structure, where v1 lives under the un-suffixed module root and v2 uses a `/v2` module path, which is also consistent with Go’s module versioning model for major versions v2 and higher. SemVer also explicitly supports prerelease identifiers like `-alpha` and `-beta`. ([Infoblox Open][2])

The important part is this:

**Do not put alpha/beta in the import path by default.**
Put alpha/beta in the **release version** and optionally in **lifecycle metadata**, not in the path.

That gives you:

* stable import/package identity for one API line
* ability to publish prereleases
* no forced import rewrites between alpha → beta → GA
* a clean rule that only **breaking changes** create a new import path

---

# 1) How APX would need to implement it

## A. Define one canonical identity model

APX needs first-class fields for these concepts:

* `api_id`
  example: `proto/payments/ledger/v1`

* `api_line`
  example: `v1`

* `release_version`
  example: `v1.0.0-beta.1`

* `lifecycle`
  example: `beta`

* `source_repo`
  example: `github.com/acme/apis`

* `language_coordinates`

  * Go module path
  * Go import path
  * PyPI package name
  * Maven coordinates
  * etc.

Right now the public config model in code is still very flat, with fields like `org`, `repo`, `module_roots`, and language targets, but it does not clearly represent separate API identity, source identity, and language distribution identity. ([GitHub][3])

So APX should explicitly model:

```yaml
api:
  id: proto/payments/ledger/v1
  format: proto
  domain: payments
  name: ledger
  line: v1
  lifecycle: beta

source:
  repo: github.com/acme/apis
  path: proto/payments/ledger/v1

releases:
  current: v1.0.0-beta.1

languages:
  go:
    module: github.com/acme/apis/proto/payments/ledger
    import: github.com/acme/apis/proto/payments/ledger/v1
```

That makes the identity system explicit instead of implicit.

## B. Make one repo the default truth

APX docs currently describe the canonical repo as the single source of truth. I would lean into that and remove the ambiguous `apis-go` story from the default model. ([Infoblox Open][1])

Recommended rule:

* **default:** one canonical repo, `apis`
* **optional later:** separate generated-package repos, but only as an explicit advanced feature

If APX ever wants `apis-go`, it must be represented as an explicit second distribution target, not something users infer from examples.

## C. Separate API line versioning from release versioning

APX should treat these as different things:

* **API line** = compatibility namespace
  `v1`, `v2`

* **release version** = semver release of that line
  `v1.0.0-alpha.1`, `v1.0.0`, `v1.1.0`, `v1.1.1`

This gives APX a stable rule set:

* additive changes within `v1` → stay on `proto/payments/ledger/v1`
* breaking changes → new API line `proto/payments/ledger/v2`
* alpha/beta/rc/ga → semver prerelease or normal release on the same API line

That aligns well with APX’s current documented publishing flow using subdirectory/version tags and with SemVer’s prerelease format. ([Infoblox Open][4])

## D. Make lifecycle metadata first-class

APX should not make users infer maturity from tags alone.

Catalog entries should include something like:

```yaml
apis:
  - id: proto/payments/ledger/v1
    latest_release: v1.0.0-beta.1
    lifecycle: beta
    owners:
      - team-payments
```

This is important because:

* `v1.0.0-beta.1` is a release artifact
* `beta` is a product signal
* `deprecated` or `sunset` are governance signals

Those should be queryable in catalog/search.

## E. Enforce path derivation rules in codegen and publish

APX should derive paths from a single algorithm, not from examples.

For Go, the rules should be:

### For v1

* API ID: `proto/payments/ledger/v1`
* module path: `github.com/acme/apis/proto/payments/ledger`
* import path: `github.com/acme/apis/proto/payments/ledger/v1`

### For v2+

* API ID: `proto/payments/ledger/v2`
* module path: `github.com/acme/apis/proto/payments/ledger/v2`
* import path: `github.com/acme/apis/proto/payments/ledger/v2`

APX should compute these automatically and validate them against `go_package`, generated code, and publish artifacts. The canonical repo structure docs already imply this distinction for v1 vs v2 modules. ([Infoblox Open][2])

## F. Publish should generate and validate canonical coordinates

When a user runs publish, APX should:

1. read the API ID
2. derive canonical source path
3. derive language coordinates
4. validate `go_package` and module path
5. create or validate the canonical `go.mod`
6. create the subdirectory tag
7. record lifecycle/version in catalog

The docs already say publish can add canonical `go.mod` during publication and that canonical CI is responsible for official releases. That makes this the right place to enforce identity rules. ([Infoblox Open][4])

## G. Add identity inspection commands

APX needs commands that explain what it thinks the identity is.

Examples:

```bash
apx inspect identity proto/payments/ledger/v1
apx inspect release proto/payments/ledger/v1@v1.0.0-beta.1
apx explain go-path proto/payments/ledger/v1
```

Those commands should print:

* API ID
* source repo path
* module path
* import path
* lifecycle
* latest stable
* latest prerelease

That would make the model debuggable.

---

# 2) How developers would use it

## Example: authoring a brand new API

A team starts an API:

```text
proto/payments/ledger/v1
```

Their proto file contains:

```proto
syntax = "proto3";

package acme.payments.ledger.v1;
option go_package = "github.com/acme/apis/proto/payments/ledger/v1;ledgerv1";
```

They run:

```bash
apx lint proto/payments/ledger/v1
apx breaking proto/payments/ledger/v1
```

Then publish an early build:

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental
```

Later:

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta
```

Then GA:

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable
```

### What does not change?

The import path:

```go
import ledgerv1 "github.com/acme/apis/proto/payments/ledger/v1"
```

That stability is the key benefit.

## Example: consuming the beta

A developer wants the beta:

```bash
go get github.com/acme/apis/proto/payments/ledger@v1.0.0-beta.1
```

Then in code:

```go
import ledgerv1 "github.com/acme/apis/proto/payments/ledger/v1"
```

Later, when GA arrives:

```bash
go get github.com/acme/apis/proto/payments/ledger@v1.0.0
```

Their import path stays the same. Only the resolved module version changes.

## Example: additive change in v1

A new optional field is added. That should remain on the same API line.

Release:

```bash
apx publish proto/payments/ledger/v1 --version v1.1.0 --lifecycle stable
```

Consumers upgrade with no import rewrite:

```bash
go get github.com/acme/apis/proto/payments/ledger@v1.1.0
```

## Example: breaking change

A breaking field rename or response change happens.

That must become a new API line:

```text
proto/payments/ledger/v2
```

New proto:

```proto
package acme.payments.ledger.v2;
option go_package = "github.com/acme/apis/proto/payments/ledger/v2;ledgerv2";
```

Early release:

```bash
apx publish proto/payments/ledger/v2 --version v2.0.0-alpha.1 --lifecycle experimental
```

Consumer code explicitly opts into the break:

```go
import ledgerv2 "github.com/acme/apis/proto/payments/ledger/v2"
```

That is exactly the kind of churn you *want* only on breaking changes.

## Example: search and discovery

A developer should be able to run:

```bash
apx search ledger
apx show proto/payments/ledger/v1
```

And see something like:

```text
API: proto/payments/ledger/v1
Lifecycle: beta
Latest stable: none
Latest prerelease: v1.0.0-beta.1
Go module: github.com/acme/apis/proto/payments/ledger
Go import: github.com/acme/apis/proto/payments/ledger/v1
Owners: team-payments
```

That makes adoption easy and removes guesswork.

---

## Bottom line

The path forward I’d choose is:

* **one canonical repo by default**
* **one repo-neutral API ID**
* **major compatibility in the path** (`v1`, `v2`)
* **alpha/beta/rc in semver release tags**
* **lifecycle as explicit metadata**
* **derived language-specific coordinates**
* **no separate `apis-go` default story**

That gives APX a model that is simple, versionable, and understandable.

The slogan version is:

> **Path = compatibility**
> **Tag = release version**
> **Lifecycle = support signal**

That is the cleanest way to make APX’s identity model trustworthy.

I can turn this next into a Spec Kit feature spec for “Canonical Identity and Versioning Model,” with acceptance criteria and migration rules.

[1]: https://infobloxopen.github.io/apx/getting-started/quickstart.html?utm_source=chatgpt.com "Quick Start — APX Documentation - infobloxopen.github.io"
[2]: https://infobloxopen.github.io/apx/canonical-repo/structure.html?utm_source=chatgpt.com "Canonical Repository Structure — APX Documentation"
[3]: https://raw.githubusercontent.com/Infobloxopen/apx/main/apx.example.yaml "raw.githubusercontent.com"
[4]: https://infobloxopen.github.io/apx/publishing/?utm_source=chatgpt.com "Publishing Workflow — APX Documentation"
