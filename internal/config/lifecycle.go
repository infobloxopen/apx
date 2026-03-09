package config

import (
	"fmt"
	"strings"
)

// Lifecycle constants define the canonical API lifecycle states.
//
// The lifecycle is a support/stability signal independent of version numbers:
//
//	experimental — early exploration; no compatibility guarantee
//	beta         — API surface is stabilizing; breaking changes still possible
//	stable       — full backward compatibility within the major line
//	deprecated   — maintained but no new features; consumers should migrate
//	sunset       — end of life; no further releases permitted
//
// "preview" is accepted as a backward-compatible alias for "beta".
const (
	LifecycleExperimental = "experimental"
	LifecycleBeta         = "beta"
	LifecyclePreview      = "preview" // alias for beta (backward compat)
	LifecycleStable       = "stable"
	LifecycleDeprecated   = "deprecated"
	LifecycleSunset       = "sunset"
)

// NormalizeLifecycle converts backward-compatible lifecycle aliases to their
// canonical form. "preview" → "beta"; all others pass through unchanged.
func NormalizeLifecycle(lifecycle string) string {
	if lifecycle == LifecyclePreview {
		return LifecycleBeta
	}
	return lifecycle
}

// ValidateVersionLifecycle checks that a version's prerelease tag is
// compatible with the declared lifecycle state.
//
// Rules:
//
//	experimental → must have -alpha.* prerelease
//	beta (or preview alias) → must have -alpha.*, -beta.*, or -rc.* prerelease
//	stable       → must NOT have a prerelease tag
//	deprecated   → any version allowed (warning emitted by caller)
//	sunset       → blocked unless override is set
//
// Returns nil if the combination is legal.
func ValidateVersionLifecycle(version, lifecycle string) error {
	if lifecycle == "" {
		return nil // No lifecycle declared — skip enforcement.
	}

	lc := NormalizeLifecycle(lifecycle)
	pre := extractPrerelease(version)

	switch lc {
	case LifecycleExperimental:
		if !strings.HasPrefix(pre, "alpha") {
			return fmt.Errorf(
				"lifecycle %q requires an -alpha.* prerelease tag, got version %q",
				lifecycle, version,
			)
		}
	case LifecycleBeta:
		if !isBetaPrerelease(pre) {
			return fmt.Errorf(
				"lifecycle %q requires a prerelease tag (-alpha.*, -beta.*, or -rc.*), got version %q",
				lifecycle, version,
			)
		}
	case LifecycleStable:
		if pre != "" {
			return fmt.Errorf(
				"lifecycle %q does not allow prerelease tags, got version %q",
				lifecycle, version,
			)
		}
	case LifecycleDeprecated:
		// Allowed but caller should warn.
		return nil
	case LifecycleSunset:
		return fmt.Errorf(
			"lifecycle %q blocks new releases; use --force to override",
			lifecycle,
		)
	default:
		return fmt.Errorf("unknown lifecycle %q", lifecycle)
	}

	return nil
}

// isBetaPrerelease returns true if the prerelease tag is one of
// the allowed beta-phase labels: alpha, beta, or rc.
func isBetaPrerelease(pre string) bool {
	return strings.HasPrefix(pre, "alpha") ||
		strings.HasPrefix(pre, "beta") ||
		strings.HasPrefix(pre, "rc")
}

// LifecycleAllowsRelease returns true if the lifecycle allows any release at all.
// "sunset" is the only lifecycle that blocks by default.
func LifecycleAllowsRelease(lifecycle string) bool {
	return NormalizeLifecycle(lifecycle) != LifecycleSunset
}

// LifecycleRequiresWarning returns true if releasing under this lifecycle
// should emit a deprecation warning.
func LifecycleRequiresWarning(lifecycle string) bool {
	return NormalizeLifecycle(lifecycle) == LifecycleDeprecated
}

// lifecycleOrder defines the canonical progression of lifecycle states.
// A lifecycle can only move forward (to a higher index) in this list,
// never backward.
var lifecycleOrder = map[string]int{
	"experimental": 0,
	"preview":      1,
	"beta":         1, // canonical; preview is a backward-compat alias
	"stable":       2,
	"deprecated":   3,
	"sunset":       4,
}

// ValidateLifecycleTransition checks that moving from one lifecycle state
// to another is legal. The only legal direction is forward:
//
//	experimental → beta → stable → deprecated → sunset
//
// Transitions backward (e.g. stable → experimental) are always illegal.
// Staying at the same state is always legal.
// An empty "from" is treated as a fresh API (any target is legal).
func ValidateLifecycleTransition(from, to string) error {
	if from == "" {
		// First-time lifecycle assignment — any target is legal.
		return nil
	}

	nFrom := NormalizeLifecycle(from)
	nTo := NormalizeLifecycle(to)

	if nFrom == nTo {
		return nil
	}

	fromIdx, fromOK := lifecycleOrder[nFrom]
	toIdx, toOK := lifecycleOrder[nTo]
	if !fromOK {
		return fmt.Errorf("unknown source lifecycle %q", from)
	}
	if !toOK {
		return fmt.Errorf("unknown target lifecycle %q", to)
	}

	if toIdx < fromIdx {
		return fmt.Errorf(
			"illegal lifecycle transition %s → %s: lifecycle can only move forward (experimental → beta → stable → deprecated → sunset)",
			from, to,
		)
	}

	// Forward transitions that skip states are allowed (e.g. experimental → stable).
	return nil
}

// ---------------------------------------------------------------------------
// v0 line policy
// ---------------------------------------------------------------------------

// ValidateV0Lifecycle enforces that a v0 API line uses only experimental or
// beta lifecycle states. v0 lines are inherently unstable and must signal
// that to consumers.
func ValidateV0Lifecycle(lifecycle string) error {
	lc := NormalizeLifecycle(lifecycle)
	if lc != LifecycleExperimental && lc != LifecycleBeta {
		return fmt.Errorf(
			"v0 API lines require lifecycle %q or %q, got %q",
			LifecycleExperimental, LifecycleBeta, lifecycle,
		)
	}
	return nil
}

// V0AllowsBreaking returns true because v0 lines always allow breaking
// changes (no backward-compatibility guarantee).
func V0AllowsBreaking() bool {
	return true
}

// ---------------------------------------------------------------------------
// Compatibility promise & production-use recommendation
// ---------------------------------------------------------------------------

// CompatibilityPromise describes the backward-compatibility contract for an
// API given its line and lifecycle state.
type CompatibilityPromise struct {
	Level          string // "none", "stabilizing", "full", "maintenance", "eol"
	Summary        string // one-line human-readable summary
	BreakingPolicy string // what the breaking-change rule is
}

// DeriveCompatibilityPromise computes the compatibility promise from an API's
// version line and lifecycle state.
func DeriveCompatibilityPromise(line, lifecycle string) CompatibilityPromise {
	isV0 := strings.TrimPrefix(line, "v") == "0"
	lc := NormalizeLifecycle(lifecycle)

	switch {
	case isV0 || lc == LifecycleExperimental:
		return CompatibilityPromise{
			Level:          "none",
			Summary:        "No compatibility guarantee — breaking changes expected",
			BreakingPolicy: "Breaking changes allowed in any release",
		}
	case lc == LifecycleBeta:
		return CompatibilityPromise{
			Level:          "stabilizing",
			Summary:        "API surface is stabilizing — minor breaking changes possible",
			BreakingPolicy: "Breaking changes require a new prerelease series",
		}
	case lc == LifecycleStable:
		return CompatibilityPromise{
			Level:          "full",
			Summary:        "Full backward compatibility within the major line",
			BreakingPolicy: "Breaking changes require a new major API line",
		}
	case lc == LifecycleDeprecated:
		return CompatibilityPromise{
			Level:          "maintenance",
			Summary:        "Maintained for security/critical fixes — no new features",
			BreakingPolicy: "No breaking changes; migrate to successor line",
		}
	case lc == LifecycleSunset:
		return CompatibilityPromise{
			Level:          "eol",
			Summary:        "End of life — no further releases",
			BreakingPolicy: "N/A — releases are blocked",
		}
	default:
		return CompatibilityPromise{
			Level:          "unknown",
			Summary:        "Lifecycle not declared",
			BreakingPolicy: "Unknown",
		}
	}
}

// ProductionRecommendation returns a human-readable recommendation about
// whether this API should be used in production.
func ProductionRecommendation(lifecycle string) string {
	lc := NormalizeLifecycle(lifecycle)
	switch lc {
	case LifecycleExperimental:
		return "Not recommended for production use"
	case LifecycleBeta:
		return "Use with caution — API may change before stable release"
	case LifecycleStable:
		return "Recommended for production use"
	case LifecycleDeprecated:
		return "Migrate away — maintenance-only, no new features"
	case LifecycleSunset:
		return "Do not use — end of life"
	default:
		return "Unknown lifecycle — check API documentation"
	}
}

// extractPrerelease extracts the prerelease portion of a semver string.
// "v1.2.3-beta.1" → "beta.1", "v1.2.3" → ""
func extractPrerelease(version string) string {
	v := strings.TrimPrefix(version, "v")

	// Strip build metadata first (everything after +)
	if idx := strings.Index(v, "+"); idx >= 0 {
		v = v[:idx]
	}

	// Find the prerelease separator
	if idx := strings.Index(v, "-"); idx >= 0 {
		return v[idx+1:]
	}
	return ""
}
