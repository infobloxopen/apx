package config

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// BumpKind describes the type of version bump.
type BumpKind string

const (
	BumpPatch BumpKind = "PATCH"
	BumpMinor BumpKind = "MINOR"
	BumpMajor BumpKind = "MAJOR"
	BumpNone  BumpKind = "NONE"
)

// VersionSuggestion is the structured output of the semver suggest engine.
type VersionSuggestion struct {
	Current   string   `json:"current"`   // Latest existing version (empty if none)
	Suggested string   `json:"suggested"` // Full suggested version string
	Bump      BumpKind `json:"bump"`      // Kind of bump applied
	Reasoning string   `json:"reasoning"` // Human-readable explanation
}

// SemVer represents a parsed semantic version.
type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string // e.g. "alpha.1", "beta.2"
	Build      string // e.g. "build.123"
	Raw        string // original string
}

// semverRegex matches v-prefixed semver strings.
var semverRegex = regexp.MustCompile(
	`^v?(\d+)\.(\d+)\.(\d+)` +
		`(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?` +
		`(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`,
)

// ParseSemVer parses a version string into its components.
func ParseSemVer(version string) (*SemVer, error) {
	m := semverRegex.FindStringSubmatch(version)
	if m == nil {
		return nil, fmt.Errorf("invalid semver %q", version)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])

	return &SemVer{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: m[4],
		Build:      m[5],
		Raw:        version,
	}, nil
}

// String renders the SemVer as a v-prefixed string.
func (sv *SemVer) String() string {
	s := fmt.Sprintf("v%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
	if sv.Prerelease != "" {
		s += "-" + sv.Prerelease
	}
	if sv.Build != "" {
		s += "+" + sv.Build
	}
	return s
}

// IsPrerelease returns true if the version has a prerelease tag.
func (sv *SemVer) IsPrerelease() bool {
	return sv.Prerelease != ""
}

// CompareSemVer returns -1, 0, or 1 for ordering (ignoring build metadata).
func CompareSemVer(a, b *SemVer) int {
	if a.Major != b.Major {
		return cmpInt(a.Major, b.Major)
	}
	if a.Minor != b.Minor {
		return cmpInt(a.Minor, b.Minor)
	}
	if a.Patch != b.Patch {
		return cmpInt(a.Patch, b.Patch)
	}
	// Pre-release versions have lower precedence than the release version.
	if a.Prerelease == "" && b.Prerelease != "" {
		return 1
	}
	if a.Prerelease != "" && b.Prerelease == "" {
		return -1
	}
	if a.Prerelease != b.Prerelease {
		return comparePrerelease(a.Prerelease, b.Prerelease)
	}
	return 0
}

// comparePrerelease compares dot-separated prerelease identifiers per semver spec.
func comparePrerelease(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		ai, aErr := strconv.Atoi(as[i])
		bi, bErr := strconv.Atoi(bs[i])
		switch {
		case aErr == nil && bErr == nil:
			if ai != bi {
				return cmpInt(ai, bi)
			}
		case aErr == nil:
			return -1 // numeric < string
		case bErr == nil:
			return 1
		default:
			if as[i] < bs[i] {
				return -1
			}
			if as[i] > bs[i] {
				return 1
			}
		}
	}
	return cmpInt(len(as), len(bs))
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// LatestVersion finds the highest semver version from a list of version strings.
// Only versions whose major matches lineMajor are considered.
// Returns ("", nil) if no matching versions are found.
func LatestVersion(versions []string, lineMajor int) (string, error) {
	var best *SemVer
	for _, v := range versions {
		sv, err := ParseSemVer(v)
		if err != nil {
			continue // skip unparseable versions
		}
		if sv.Major != lineMajor {
			continue
		}
		if best == nil || CompareSemVer(sv, best) > 0 {
			best = sv
		}
	}
	if best == nil {
		return "", nil
	}
	return best.String(), nil
}

// LatestVersionFromTags extracts versions from git-style tags with a prefix.
// Tag format: "<prefix>/v<semver>" (e.g. "proto/payments/ledger/v1/v1.2.3")
// The prefix is stripped to extract just the version part.
func LatestVersionFromTags(tags []string, tagPrefix string, lineMajor int) (string, error) {
	var versions []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !strings.HasPrefix(tag, tagPrefix+"/") {
			continue
		}
		ver := strings.TrimPrefix(tag, tagPrefix+"/")
		versions = append(versions, ver)
	}
	return LatestVersion(versions, lineMajor)
}

// SortVersions sorts a slice of version strings in ascending semver order.
func SortVersions(versions []string) []string {
	type parsed struct {
		raw string
		sv  *SemVer
	}
	var items []parsed
	for _, v := range versions {
		sv, err := ParseSemVer(v)
		if err != nil {
			continue
		}
		items = append(items, parsed{raw: v, sv: sv})
	}
	sort.Slice(items, func(i, j int) bool {
		return CompareSemVer(items[i].sv, items[j].sv) < 0
	})
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.raw
	}
	return out
}

// ValidateVersionLine checks that a version's major component matches the
// API line's major version. For example, v1.2.3 is valid for line "v1" but
// not for line "v2". v0 lines match major version 0.
func ValidateVersionLine(version, line string) error {
	sv, err := ParseSemVer(version)
	if err != nil {
		return fmt.Errorf("invalid version %q: %w", version, err)
	}
	major, err := LineMajor(line)
	if err != nil {
		return err
	}
	if sv.Major != major {
		return fmt.Errorf(
			"version %q has major %d but API line %q requires major %d",
			version, sv.Major, line, major,
		)
	}
	return nil
}

// SuggestVersion determines the recommended version bump based on whether
// breaking changes were detected, the current latest version, and the
// lifecycle state.
//
// Rules:
//   - v0 line: breaking changes are allowed (bump minor), non-breaking → minor or patch
//   - v1+ breaking change detected → reject (return error, caller must create new major line)
//   - non-breaking additive change → minor
//   - bugfix/docs/generator-only change → patch (caller signals via hasChanges=false)
//   - if no current version exists → initial version for the line (e.g. v0.1.0, v1.0.0, v2.0.0)
//
// The lifecycle parameter is used to attach the correct prerelease tag:
//   - "experimental" → -alpha.1
//   - "preview"/"beta" → -beta.1
//   - "stable" → no prerelease
//   - "deprecated" → no prerelease (but caller should warn)
//   - "sunset" → blocked
func SuggestVersion(current string, hasBreaking, hasChanges bool, lifecycle, line string) (*VersionSuggestion, error) {
	// Sunset blocks completely
	if NormalizeLifecycle(lifecycle) == "sunset" {
		return nil, fmt.Errorf("lifecycle %q blocks new releases; create a new API line or use --force", lifecycle)
	}

	major, err := LineMajor(line)
	if err != nil {
		return nil, err
	}

	isV0 := major == 0

	// Determine prerelease suffix based on lifecycle
	preSuffix := lifecyclePrerelease(lifecycle, 1)

	// Breaking changes
	if hasBreaking {
		if isV0 {
			// v0 allows breaking changes — bump minor
			if current == "" {
				suggested := "v0.1.0"
				if preSuffix != "" {
					suggested += "-" + preSuffix
				}
				return &VersionSuggestion{
					Current:   "",
					Suggested: suggested,
					Bump:      BumpMinor,
					Reasoning: "Breaking changes on v0 line — minor bump (v0 has no compatibility guarantee)",
				}, nil
			}
			sv, parseErr := ParseSemVer(current)
			if parseErr != nil {
				return nil, fmt.Errorf("cannot parse current version %q: %w", current, parseErr)
			}
			suggested := fmt.Sprintf("v0.%d.0", sv.Minor+1)
			if preSuffix != "" {
				suggested += "-" + preSuffix
			}
			return &VersionSuggestion{
				Current:   current,
				Suggested: suggested,
				Bump:      BumpMinor,
				Reasoning: "Breaking changes on v0 line — minor bump (v0 has no compatibility guarantee)",
			}, nil
		}

		// v1+ — breaking changes are not allowed within a line
		return &VersionSuggestion{
				Current:   current,
				Suggested: "",
				Bump:      BumpMajor,
				Reasoning: fmt.Sprintf(
					"Breaking changes detected. Cannot bump within line %s (major %d). "+
						"Create a new API line (e.g. v%d) and start fresh.",
					line, major, major+1,
				),
			}, fmt.Errorf(
				"breaking changes detected; cannot release on line %s — create a new API line v%d",
				line, major+1,
			)
	}

	// No current version → initial release
	if current == "" {
		suggested := fmt.Sprintf("v%d.0.0", major)
		if preSuffix != "" {
			suggested += "-" + preSuffix
		}
		return &VersionSuggestion{
			Current:   "",
			Suggested: suggested,
			Bump:      BumpMinor,
			Reasoning: fmt.Sprintf("Initial release for line %s", line),
		}, nil
	}

	// Parse current version
	sv, err := ParseSemVer(current)
	if err != nil {
		return nil, fmt.Errorf("cannot parse current version %q: %w", current, err)
	}

	// If current is a prerelease, bump the prerelease counter
	if sv.IsPrerelease() {
		next, reason := bumpPrerelease(sv, lifecycle)
		return &VersionSuggestion{
			Current:   current,
			Suggested: next,
			Bump:      BumpMinor,
			Reasoning: reason,
		}, nil
	}

	// Determine bump type for stable versions
	var bump BumpKind
	var reason string
	var next SemVer

	if hasChanges {
		bump = BumpMinor
		reason = "Non-breaking additive changes detected"
		next = SemVer{Major: sv.Major, Minor: sv.Minor + 1, Patch: 0}
	} else {
		bump = BumpPatch
		reason = "Bugfix, docs, or generator-only changes"
		next = SemVer{Major: sv.Major, Minor: sv.Minor, Patch: sv.Patch + 1}
	}

	suggested := next.String()
	if preSuffix != "" {
		suggested += "-" + preSuffix
	}

	return &VersionSuggestion{
		Current:   current,
		Suggested: suggested,
		Bump:      bump,
		Reasoning: reason,
	}, nil
}

// lifecyclePrerelease returns the prerelease suffix for the given lifecycle state.
// The counter parameter is appended (e.g. "alpha.1", "beta.2").
func lifecyclePrerelease(lifecycle string, counter int) string {
	lc := NormalizeLifecycle(lifecycle)
	switch lc {
	case "experimental":
		return fmt.Sprintf("alpha.%d", counter)
	case "beta":
		return fmt.Sprintf("beta.%d", counter)
	default:
		return ""
	}
}

// bumpPrerelease increments a prerelease version intelligently.
// If the prerelease matches the lifecycle, it increments the counter.
// If the lifecycle has changed, it resets to the new lifecycle prefix.
func bumpPrerelease(sv *SemVer, lifecycle string) (string, string) {
	// Extract the prerelease prefix and counter
	parts := strings.Split(sv.Prerelease, ".")
	prefix := parts[0]
	counter := 1
	if len(parts) > 1 {
		if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
			counter = n
		}
	}

	expectedPrefix := lifecyclePrereleasePrefix(lifecycle)

	// If lifecycle matches, increment counter
	if prefix == expectedPrefix || expectedPrefix == "" {
		counter++
		next := SemVer{Major: sv.Major, Minor: sv.Minor, Patch: sv.Patch}
		next.Prerelease = fmt.Sprintf("%s.%d", prefix, counter)
		return next.String(), fmt.Sprintf("Incremented %s prerelease counter to %d", prefix, counter)
	}

	// Lifecycle changed (e.g. alpha → beta), reset counter
	next := SemVer{Major: sv.Major, Minor: sv.Minor, Patch: sv.Patch}
	next.Prerelease = fmt.Sprintf("%s.1", expectedPrefix)
	return next.String(), fmt.Sprintf("Transitioned prerelease from %s to %s", prefix, expectedPrefix)
}

// lifecyclePrereleasePrefix returns the prerelease prefix for a lifecycle.
func lifecyclePrereleasePrefix(lifecycle string) string {
	lc := NormalizeLifecycle(lifecycle)
	switch lc {
	case "experimental":
		return "alpha"
	case "beta":
		return "beta"
	default:
		return ""
	}
}

// FormatSuggestionReport produces a human-readable report of a version
// suggestion.
func FormatSuggestionReport(s *VersionSuggestion) string {
	var sb strings.Builder
	if s.Current != "" {
		sb.WriteString(fmt.Sprintf("Current version:   %s\n", s.Current))
	} else {
		sb.WriteString("Current version:   (none — first release)\n")
	}
	if s.Suggested != "" {
		sb.WriteString(fmt.Sprintf("Suggested version: %s\n", s.Suggested))
	} else {
		sb.WriteString("Suggested version: (blocked — see reasoning)\n")
	}
	sb.WriteString(fmt.Sprintf("Bump type:         %s\n", s.Bump))
	sb.WriteString(fmt.Sprintf("Reasoning:         %s\n", s.Reasoning))
	return sb.String()
}
