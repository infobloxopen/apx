package config

import (
	"fmt"
	"strings"
)

// ValidateVersionLifecycle checks that a version's prerelease tag is
// compatible with the declared lifecycle state.
//
// Rules:
//
//	experimental → must have -alpha.* prerelease
//	beta         → must have -beta.* prerelease
//	stable       → must NOT have a prerelease tag
//	deprecated   → any version allowed (warning emitted by caller)
//	sunset       → blocked unless override is set
//
// Returns nil if the combination is legal.
func ValidateVersionLifecycle(version, lifecycle string) error {
	if lifecycle == "" {
		return nil // No lifecycle declared — skip enforcement.
	}

	pre := extractPrerelease(version)

	switch lifecycle {
	case "experimental":
		if !strings.HasPrefix(pre, "alpha") {
			return fmt.Errorf(
				"lifecycle %q requires an -alpha.* prerelease tag, got version %q",
				lifecycle, version,
			)
		}
	case "beta":
		if !strings.HasPrefix(pre, "beta") {
			return fmt.Errorf(
				"lifecycle %q requires a -beta.* prerelease tag, got version %q",
				lifecycle, version,
			)
		}
	case "stable":
		if pre != "" {
			return fmt.Errorf(
				"lifecycle %q does not allow prerelease tags, got version %q",
				lifecycle, version,
			)
		}
	case "deprecated":
		// Allowed but caller should warn.
		return nil
	case "sunset":
		return fmt.Errorf(
			"lifecycle %q blocks new releases; use --force to override",
			lifecycle,
		)
	default:
		return fmt.Errorf("unknown lifecycle %q", lifecycle)
	}

	return nil
}

// LifecycleAllowsRelease returns true if the lifecycle allows any release at all.
// "sunset" is the only lifecycle that blocks by default.
func LifecycleAllowsRelease(lifecycle string) bool {
	return lifecycle != "sunset"
}

// LifecycleRequiresWarning returns true if publishing under this lifecycle
// should emit a deprecation warning.
func LifecycleRequiresWarning(lifecycle string) bool {
	return lifecycle == "deprecated"
}

// lifecycleOrder defines the canonical progression of lifecycle states.
// A lifecycle can only move forward (to a higher index) in this list,
// never backward.
var lifecycleOrder = map[string]int{
	"experimental": 0,
	"beta":         1,
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
	if from == to {
		return nil
	}

	fromIdx, fromOK := lifecycleOrder[from]
	toIdx, toOK := lifecycleOrder[to]
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
