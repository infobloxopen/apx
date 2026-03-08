Gap 5 is the difference between **having a publish concept** and having a **release system teams can trust**.

## What is wrong

APX’s intended release model is strong on paper: validate locally, tag in the app repo, run `apx publish`, then let canonical CI validate, tag, and publish packages. The docs describe that flow clearly, including subtree-based publishing and CI-owned canonical releases. But the same docs also say the step-by-step guides and full per-command docs are still “in progress,” and the only public GitHub release today is `v0.1.0-alpha.1` marked as a pre-release. That means the release workflow is still visibly early, even if the high-level design is sound. ([Infoblox Open][1])

The deeper issue is that APX’s release workflow is **multi-stage and stateful**, but the product surface does not yet make that state explicit enough. A real publish involves at least these states: authoring, validation, version selection, tag creation, publish request, canonical repo change, canonical validation, canonical tag creation, package publication, and failure recovery. Today, the docs show pieces of that flow, but they still leave important operational details under-specified, especially around retries, idempotency, partial failure, and how developers inspect release state after something goes wrong. The troubleshooting docs, for example, tell users to retry publish and resolve canonical merge conflicts manually. ([Infoblox Open][1])

There is also a product-shape problem: APX presents release-adjacent commands like `publish` and `semver suggest`, but adjacent dependency automation such as `apx update` and `apx upgrade` is still described in the FAQ as future work. That matters because a trustworthy release system is not just “how to publish,” but also “how others adopt what was published.” If the downstream half is incomplete, the upstream publish story still feels unfinished. ([Infoblox Open][2])

## Why it needs to be fixed

This is the most important gap because **release is where APX becomes real**.

If authoring is good but release is shaky, teams will keep using ad hoc scripts, hand-made GitHub Actions, and tribal knowledge for the last mile. At that point APX becomes a helper around schemas, not the platform for publishing them. Since APX’s stated value proposition is canonical distribution through a single GitHub repo with CI-only releases, the release workflow is not an optional feature; it is the center of the product. ([Infoblox Open][3])

It also needs to be fixed because release problems create organizational distrust quickly. A schema tool can survive minor DX issues, but if publishing is unreliable, teams stop betting on it. The failure modes are serious: wrong version bump, broken canonical tag, packages published from inconsistent inputs, merge conflicts during publish, or consumers not knowing what to adopt. Once people think “APX publish is flaky,” they will bypass it. ([Infoblox Open][1])

There is also a governance reason. APX is explicitly trying to balance team autonomy in app repos with centralized governance in the canonical repo. That balance only works if publish automation is deterministic and auditable. Otherwise you get the worst of both worlds: extra process plus manual intervention. ([Infoblox Open][1])

## How to go about fixing it

### 1. Define the release state machine explicitly

APX should stop treating publish as one command and instead model it as a **state machine**.

A release should have explicit states like:

* draft
* validated
* version-selected
* app-tagged
* publish-prepared
* canonical-pr-open
* canonical-validated
* canonical-released
* package-published
* failed

That gives APX a way to explain what happened, resume safely, and avoid duplicate or partial releases.

### 2. Make versioning deterministic before publish

Before APX can have a trustworthy publish flow, it needs a reliable answer to:

* what version is being released
* why that version is correct
* whether it is alpha, beta, or stable
* whether it is legal for the API line

That means `semver suggest` cannot just be a convenience command. It needs to become part of the release contract, tied to breaking-change detection and lifecycle rules.

Good rule set:

* non-breaking additive change → minor
* bugfix/docs/generator-only change → patch
* breaking change in same API line → reject, require new major line
* prerelease allowed on any unreleased or in-progress line
* lifecycle rules:

  * `experimental` can publish `-alpha.*`
  * `beta` can publish `-beta.*`
  * `stable` publishes normal SemVer
  * `deprecated` still publishes but warns
  * `sunset` blocks new releases except emergency overrides

### 3. Split publish into “prepare,” “submit,” and “finalize”

Right now APX conceptually bundles too much into “publish.” A better shape is:

* `apx release prepare`

  * validate schema
  * validate config and identity
  * compute version
  * compute lifecycle
  * build manifest of what will be published
  * show exact repo paths, tags, module paths, package coordinates

* `apx release submit`

  * create canonical branch or PR
  * attach machine-readable release manifest
  * make the operation idempotent

* `apx release finalize`

  * run in canonical CI
  * validate again
  * create official canonical tag
  * publish packages
  * update catalog
  * emit release record

That separation makes dry-run meaningful and makes failures recoverable.

### 4. Introduce a release manifest as the contract

APX needs a single machine-readable artifact that travels from app repo to canonical CI.

For example:

```yaml
release:
  api_id: proto/payments/ledger/v1
  source_repo: github.com/acme/payments-service
  source_commit: abc123
  requested_version: v1.2.0-beta.1
  lifecycle: beta
  canonical_repo: github.com/acme/apis
  canonical_path: proto/payments/ledger/v1
  go_module: github.com/acme/apis/proto/payments/ledger
  go_import: github.com/acme/apis/proto/payments/ledger/v1
  validation:
    lint: passed
    breaking: passed
    policy: passed
```

That manifest becomes the source of truth for canonical CI, audit logs, troubleshooting, and release inspection.

### 5. Make publish idempotent

This is the most important implementation property.

If a release attempt is retried, APX should detect:

* same API ID
* same source commit
* same target version
* same canonical destination

and either:

* continue safely, or
* say “this exact release already exists,” or
* say “this version exists with different contents; blocked.”

Without idempotency, retries after CI failure or merge conflict become dangerous.

### 6. Add first-class failure and recovery behavior

The current docs already acknowledge merge conflicts and retry flows. That is good, but APX should encode them, not leave them as operator folklore. ([Infoblox Open][1])

It should have explicit handling for:

* canonical repo moved since prepare
* merge conflict in subtree/copy publish
* version already taken
* policy failure in canonical CI
* generated `go.mod` mismatch
* package publication failed after canonical tag
* catalog update failed

Each should produce a precise status and next action.

### 7. Publish a release record developers can inspect

After every release, APX should make it easy to answer:

* what version was published
* from which repo and commit
* by which CI run
* with which lifecycle
* to which canonical path
* with which published language artifacts

That could be a catalog entry, a release metadata file, or both.

### 8. Treat package publication as part of release, not a side effect

The docs say canonical CI can publish packages, and the FAQ says `apx publish` can synthesize a canonical `go.mod` in the PR to the canonical repo. That is exactly why packaging must be part of the formal release design rather than an optional afterthought. Otherwise schema publication and SDK/package publication can drift. ([Infoblox Open][4])

A release should not be “done” until APX can prove the expected artifacts were published or deliberately skipped.

## The practical path forward

I would implement this in four phases.

### Phase 1: make the current flow trustworthy

* define release manifest
* make dry-run complete and human-readable
* make publish idempotent
* add explicit status/error codes
* add `apx release inspect`

### Phase 2: make versioning and lifecycle first-class

* formalize `alpha`, `beta`, `stable`, `deprecated`, `sunset`
* tie `semver suggest` to breaking analysis
* validate illegal transitions

### Phase 3: formalize canonical CI release processing

* canonical CI consumes release manifest
* re-validates and produces release record
* updates catalog and package outputs
* emits stable audit trail

### Phase 4: improve operator and consumer workflows

* release history and inspection
* dependents visibility
* safe update/upgrade flows
* promotion flows such as beta → stable without ambiguity

## Bottom line

What is wrong is not that APX lacks a publish idea. It has one. The problem is that the current public product still looks like an **early release workflow design** rather than a fully operational release system. That needs to be fixed because publishing is the core promise of APX. The way to fix it is to make release a formal, inspectable, idempotent state machine with explicit versioning, lifecycle, failure recovery, and canonical CI handoff. ([GitHub][5])

I can turn this into a Spec Kit feature with a `specify` prompt and a matching `plan` prompt next.

[1]: https://infobloxopen.github.io/apx/publishing/ "Publishing Workflow — APX Documentation"
[2]: https://infobloxopen.github.io/apx/cli-reference/index.html "CLI Reference — APX Documentation"
[3]: https://infobloxopen.github.io/apx/index.html?utm_source=chatgpt.com "APX — API Schema Management — APX Documentation"
[4]: https://infobloxopen.github.io/apx/troubleshooting/faq.html "Frequently Asked Questions — APX Documentation"
[5]: https://github.com/Infobloxopen/apx/releases "Releases · infobloxopen/apx · GitHub"
