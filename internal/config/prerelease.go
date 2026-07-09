package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Pre-release version mechanics (ARCH-271, apx#30 "A" + acceptance criteria).
//
// The develop channel publishes pre-release versions (lifecycle beta:
// vX.Y.Z-beta.N). Two guarantees back that:
//
//   AC-1 — a fail-closed ratchet: a pre-release must be strictly greater than
//          the module line's highest already-released (GA) version, so a beta
//          can never be stranded at or below a shipped GA.
//   AC-2 — every develop build encodes part of its git commit hash so the
//          version traces to a commit. The hash lives in the PRE-RELEASE
//          segment (git-describe style, g-prefixed), never in +build metadata:
//          Go modules reject +metadata in a module version, and a g-prefix keeps
//          the identifier from being all-numeric (which SemVer would strip
//          leading zeros from / Go would reject).

// shortHashRe matches a lowercase hex commit hash (abbreviated or full).
var shortHashRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// commitHashLen is how many hex characters of the commit hash to encode. Twelve
// mirrors the Go pseudo-version convention and is collision-safe in practice.
const commitHashLen = 12

// ShortCommitHash normalizes a git commit hash to the encoded form: lowercased
// and truncated to commitHashLen hex characters. It returns "" for input that
// is not a hex commit hash (so callers can treat "no usable hash" uniformly).
func ShortCommitHash(hash string) string {
	h := strings.ToLower(strings.TrimSpace(hash))
	if !shortHashRe.MatchString(h) {
		return ""
	}
	if len(h) > commitHashLen {
		h = h[:commitHashLen]
	}
	return h
}

// EncodeCommitHash appends a git-describe-style commit-hash identifier
// (".g<shorthash>") to a version's pre-release segment (AC-2). The result stays
// valid SemVer 2.0 and resolves via `go get`.
//
// Requirements and behavior:
//   - The version MUST already carry a pre-release segment (the develop channel
//     is lifecycle beta). Encoding a hash onto a stable version is rejected —
//     the hash belongs in the pre-release segment, and Go rejects +build
//     metadata in module versions.
//   - An empty/unparseable hash leaves the version unchanged (no-op), so a
//     caller without commit info degrades gracefully rather than failing.
//   - Idempotent: a version that already ends in the same ".g<shorthash>"
//     identifier is returned unchanged.
//   - Any existing +build metadata is dropped (never emitted for module tags).
func EncodeCommitHash(version, commitHash string) (string, error) {
	sv, err := ParseSemVer(version)
	if err != nil {
		return "", fmt.Errorf("invalid version %q: %w", version, err)
	}
	short := ShortCommitHash(commitHash)
	if short == "" {
		return version, nil // no usable hash — leave the version untouched
	}
	if sv.Prerelease == "" {
		return "", fmt.Errorf(
			"cannot encode a commit hash on stable version %q: the hash must live in the pre-release segment (Go rejects +build metadata in module versions)",
			version,
		)
	}
	ident := "g" + short
	// Idempotent: don't double-append the same identifier.
	if sv.Prerelease == ident || strings.HasSuffix(sv.Prerelease, "."+ident) {
		sv.Build = ""
		return sv.String(), nil
	}
	sv.Prerelease = sv.Prerelease + "." + ident
	sv.Build = "" // never emit +build metadata for a module tag
	return sv.String(), nil
}

// HighestReleased returns the highest GA (non-pre-release) version among
// versions whose major matches lineMajor, or nil when the line has no GA
// release yet. Pre-release versions are ignored — the ratchet floor is the
// shipped stable line, not other betas.
func HighestReleased(versions []string, lineMajor int) *SemVer {
	var best *SemVer
	for _, v := range versions {
		sv, err := ParseSemVer(v)
		if err != nil {
			continue
		}
		if sv.Major != lineMajor {
			continue
		}
		if sv.IsPrerelease() {
			continue
		}
		if best == nil || CompareSemVer(sv, best) > 0 {
			best = sv
		}
	}
	return best
}

// NextLegalPrerelease returns the lowest legal pre-release version above a GA
// version: the GA's patch bumped by one, tagged -beta.1. For example a GA of
// v1.1.1 yields v1.1.2-beta.1. Used to make the ratchet error actionable.
func NextLegalPrerelease(highestGA *SemVer) string {
	if highestGA == nil {
		return ""
	}
	return fmt.Sprintf("v%d.%d.%d-beta.1", highestGA.Major, highestGA.Minor, highestGA.Patch+1)
}

// AssertPrereleaseRatchet is the fail-closed AC-1 guardrail. It fails when
// candidate is a pre-release that is NOT strictly greater than the highest
// already-released (GA) version of its line, comparing against releasedVersions
// (the module line's release tags). lineMajor scopes the comparison to the
// candidate's own major line.
//
// It is a no-op — returns nil — when:
//   - candidate is a stable (non-pre-release) version (the ratchet governs
//     pre-releases only), or
//   - the line has no GA release yet (any pre-release is legal).
//
// Build metadata is ignored in the comparison (as SemVer requires). The error
// states the next legal pre-release so the caller can ratchet forward.
func AssertPrereleaseRatchet(candidate string, releasedVersions []string, lineMajor int) error {
	cand, err := ParseSemVer(candidate)
	if err != nil {
		return fmt.Errorf("invalid version %q: %w", candidate, err)
	}
	if !cand.IsPrerelease() {
		return nil
	}
	highest := HighestReleased(releasedVersions, lineMajor)
	if highest == nil {
		return nil
	}
	if CompareSemVer(cand, highest) <= 0 {
		return fmt.Errorf(
			"pre-release %s is not greater than the highest released version %s on this line; "+
				"ratchet forward to at least %s",
			cand.String(), highest.String(), NextLegalPrerelease(highest),
		)
	}
	return nil
}
