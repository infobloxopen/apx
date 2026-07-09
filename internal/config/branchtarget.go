package config

// Branch-target routing (ARCH-271, apx#30 "B").
//
// A service repo's publish workflow runs on different source branches, and each
// should open its catalog release PR against a different base branch of the
// canonical (apis) repo: the service's stable branch (main/master) targets the
// catalog's stable branch (main), while its integration branch (develop) targets
// a pre-release branch (develop). This keeps pre-release churn off the stable
// catalog branch. The mapping is configuration, not policy: it lives in
// branch_targets (apx.yaml / .apx-publish.yaml) and is fully tweakable.

// StableBaseBranch is the base branch that carries stable (GA) releases. A
// resolved base branch equal to this is the stable channel; anything else is a
// pre-release channel. It is the conventional default; the mapping that routes
// source branches here is still configurable.
const StableBaseBranch = "main"

// DefaultBranchTargets is the built-in source-branch → base-branch mapping used
// when a config does not specify branch_targets (or omits a given source
// branch). main/master publish to the stable catalog branch; develop publishes
// to the pre-release branch.
func DefaultBranchTargets() map[string]string {
	return map[string]string{
		"main":    "main",
		"master":  "main",
		"develop": "develop",
	}
}

// ResolveTargetBranch maps a service-repo source branch to the canonical-repo
// base branch its release PR should target. Precedence:
//
//  1. an explicit entry in the configured branch_targets map;
//  2. the built-in DefaultBranchTargets entry for that source branch;
//  3. StableBaseBranch ("main") as the fail-safe default.
//
// An empty sourceBranch resolves to the stable base branch, preserving the
// pre-ARCH-271 behavior (release PRs targeted "main").
func ResolveTargetBranch(sourceBranch string, configured map[string]string) string {
	if sourceBranch == "" {
		return StableBaseBranch
	}
	if base, ok := configured[sourceBranch]; ok && base != "" {
		return base
	}
	if base, ok := DefaultBranchTargets()[sourceBranch]; ok && base != "" {
		return base
	}
	return StableBaseBranch
}

// IsPrereleaseChannel reports whether a resolved base branch is a pre-release
// channel — i.e. any branch other than the stable base branch. A develop
// publish (base "develop") is a pre-release channel; a main/master publish
// (base "main") is not.
func IsPrereleaseChannel(baseBranch string) bool {
	return baseBranch != "" && baseBranch != StableBaseBranch
}
