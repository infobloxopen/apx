Yes — this gap matters a lot, and your preview question gets to the heart of it.

## The real problem

APX currently has a **release workflow** and **version tags**, but it does not appear to have a first-class, documented model for **API lifecycle state**. The docs describe a tag-in-app → PR-to-canonical → canonical-CI release flow, with subdirectory tags and `apx semver suggest`, but the publishing and CLI docs do not describe lifecycle concepts like alpha, beta, stable, deprecated, or sunset. ([Infoblox Open][1])

That means APX is currently good at answering:

* what path is this API on?
* what version tag was released?
* how does it get published?

But not good at answering:

* is this safe to depend on?
* is it a preview?
* can it break at any time?
* is it intended for production?
* is it deprecated or headed for removal?

That is the gap.

## Why that is wrong

Because **version**, **compatibility**, and **lifecycle** are not the same thing.

SemVer says major version changes communicate backward incompatibility, pre-release labels like `-alpha` and `-beta` indicate instability and lower precedence than the final release, and major version zero (`0.y.z`) is for initial development where “anything MAY change at any time” and the public API should not be considered stable. ([Semantic Versioning][2])

Those are three different signals:

* **compatibility line**: `v0`, `v1`, `v2`
* **release version**: `1.2.0`, `1.0.0-alpha.3`
* **lifecycle/support signal**: preview, stable, deprecated, sunset

If APX does not model them separately, teams will start using one to stand in for another. That causes confusion fast.

A few examples of the confusion:

### Example 1: “alpha” gets overloaded

A team publishes `v1.0.0-alpha.1`, then `v1.0.0-alpha.2`, but makes breaking changes between them. SemVer allows a pre-release to be unstable, but if APX does not clearly distinguish “preview of future v1” from “rolling unstable design work,” consumers will not know whether they are looking at a near-final contract or a moving target. ([Semantic Versioning][2])

### Example 2: consumers cannot tell what is safe

If APX only shows `proto/payments/ledger/v1@v1.0.0-beta.2`, that tells me the version string, but not the platform policy. Is beta allowed for production? Is it supported by the owning team? Is it expected to change? The version string alone is not enough.

### Example 3: deprecation becomes tribal knowledge

Without first-class lifecycle, “deprecated” lives in README text, release notes, or team memory. That makes discovery, warning, and governance weak.

## The preview question: can developers publish backward-incompatible previews?

Yes — and this is where the model needs to be more precise.

My recommendation is:

### Do not use “alpha” as the only mechanism for incompatible preview work

Use **two different mechanisms** depending on what the team is doing.

## Case A: near-final prerelease of a known API line

Use this when the team already knows the public line is going to be `v1`, and they want people to test it before GA.

Example:

* API line: `proto/payments/ledger/v1`
* releases:

  * `v1.0.0-alpha.1`
  * `v1.0.0-beta.1`
  * `v1.0.0-rc.1`
  * `v1.0.0`

This is what SemVer prereleases are for: unstable previews of an intended normal release. Pre-release versions have lower precedence than the final version, and they indicate instability. ([Semantic Versioning][2])

This is good for:

* external testing
* early adopters
* integration validation
* staged rollout before GA

This is **not** the best fit for long-running, frequently breaking design churn.

## Case B: rolling incompatible preview work

Use this when the API is still evolving rapidly and backward incompatibility is expected.

Here, the better answer is **not “alpha on v1”**. The better answer is a **`v0` compatibility line** or an explicit **preview line**.

SemVer explicitly says `0.y.z` is for initial development, anything may change at any time, and the API should not be considered stable. It also says if you are changing the API every day, you should either still be in `0.y.z` or on a separate development branch working on the next major version. ([Semantic Versioning][2])

So APX should support something like:

* API line: `proto/payments/ledger/v0`
* releases:

  * `0.1.0`
  * `0.2.0`
  * `0.3.0`
* lifecycle: `experimental`

That gives developers a clean way to publish ongoing previews that others can observe or consume, while making the risk explicit.

In other words:

* **`v0`** = unstable compatibility line, breaking changes allowed
* **`v1.0.0-alpha.N`** = prerelease of an intended stable `v1`
* **`v1.0.0`** = stable public contract

That is much clearer than forcing everything through “alpha.”

## What is wrong with APX today

The problem is not that APX cannot version APIs. It can. The docs already describe per-API tags, semantic version suggestion, and canonical CI releases. ([Infoblox Open][1])

The problem is that APX does not appear to have a documented way to express:

* this API is experimental
* this API is a preview
* this API is stable
* this API is deprecated
* this API is sunset
* this line permits incompatible churn
* this prerelease is intended to become `v1`
* this is only for observation versus supported production use

Without that, APX forces teams to encode meaning in version strings alone, which is not enough.

## How it should be fixed

The fix is to make lifecycle first-class and to separate the three axes.

## 1. Separate compatibility, release version, and lifecycle

APX should model all three explicitly.

A good model is:

```yaml
api:
  id: proto/payments/ledger/v0
  line: v0
  lifecycle: experimental

release:
  version: 0.3.0
  channel: preview
```

Or:

```yaml
api:
  id: proto/payments/ledger/v1
  line: v1
  lifecycle: preview

release:
  version: 1.0.0-beta.2
  channel: prerelease
```

And later:

```yaml
api:
  id: proto/payments/ledger/v1
  line: v1
  lifecycle: stable

release:
  version: 1.0.0
  channel: ga
```

The key is that APX should not make the user infer lifecycle from the version string.

## 2. Use better lifecycle names

I would not make `alpha` and `beta` the top-level lifecycle taxonomy.

I would use lifecycle values like:

* `experimental`
* `preview`
* `stable`
* `deprecated`
* `sunset`

Then use SemVer prerelease labels for release phase:

* `-alpha.N`
* `-beta.N`
* `-rc.N`

Why this is better:

* `experimental` tells consumers the **support/risk posture**
* `alpha` tells consumers the **release phase**
* `v0` or `v1` tells consumers the **compatibility contract**

Each one does a different job.

## 3. Add rules APX can enforce

APX should enforce policy like this:

### For `v0`

* lifecycle must be `experimental` or `preview`
* backward incompatible changes are allowed
* publish commands should warn loudly
* discovery should label it unstable

### For `v1+` prereleases

* `1.0.0-alpha.N`, `1.0.0-beta.N`, `1.0.0-rc.N` are allowed
* APX should say these are previews of the `v1` line
* breaking changes between prereleases may be allowed, but with warnings
* APX should require a clear owner and changelog for preview releases

### For stable

* normal SemVer applies
* backward incompatible changes on the same line are blocked
* deprecations create warnings and timelines

### For deprecated/sunset

* APX search/show should warn
* consumers should get warnings on add/update
* metadata should include replacement and dates

## 4. Make this visible in the catalog and CLI

When a developer runs:

```bash
apx show proto/payments/ledger/v0
```

they should see:

```text
API: proto/payments/ledger/v0
Lifecycle: experimental
Compatibility: unstable
Latest release: 0.3.0
Production use: not recommended
Owner: team-payments
```

And for a prerelease on `v1`:

```text
API: proto/payments/ledger/v1
Lifecycle: preview
Latest prerelease: 1.0.0-beta.2
Latest stable: none
Intended GA line: v1
Compatibility promise: not yet final
```

That makes consumption decisions much easier.

## 5. Give developers a supported preview workflow

This is the part you care about most.

I would make APX support **two official preview workflows**.

### Workflow 1: rolling preview line

For APIs that are still being shaped:

```bash
apx publish proto/payments/ledger/v0 --version 0.4.0 --lifecycle experimental
```

This is the right workflow when others may observe or consume it, but everyone understands it can break.

### Workflow 2: prerelease on upcoming stable line

For APIs close to launch:

```bash
apx publish proto/payments/ledger/v1 --version 1.0.0-alpha.1 --lifecycle preview
apx publish proto/payments/ledger/v1 --version 1.0.0-beta.1 --lifecycle preview
apx publish proto/payments/ledger/v1 --version 1.0.0 --lifecycle stable
```

This is the right workflow when you want preview users to test the actual `v1` contract before GA.

## My recommendation for your specific concern

For “developers can publish alpha APIs that are backward incompatible so others can observe or even consume them,” I would not make **alpha** the main answer.

I would make the official APX answer:

* use **`v0` + `experimental`** for rolling incompatible preview development
* use **`v1.0.0-alpha.N` / `beta.N` / `rc.N`** for prereleases of a mostly-defined upcoming stable line

That is cleaner, easier to explain, and closer to how SemVer already distinguishes initial development from prereleases of a known release. ([Semantic Versioning][2])

## Bottom line

What is wrong is that APX currently has versioning and publishing, but not a first-class lifecycle model. That is wrong because developers and consumers need separate answers for compatibility, release phase, and support level. The fix is to model those separately and give teams two explicit preview paths: **`v0` experimental lines for rolling incompatible work**, and **alpha/beta/rc prereleases for near-final lines**. ([Infoblox Open][1])


[1]: https://infobloxopen.github.io/apx/publishing/index.html "Publishing Workflow — APX Documentation"
[2]: https://semver.org/ "Semantic Versioning 2.0.0 | Semantic Versioning"
