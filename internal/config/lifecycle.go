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
