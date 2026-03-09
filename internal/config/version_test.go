package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ParseSemVer
// ---------------------------------------------------------------------------

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		wantPatch int
		wantPre   string
		wantBuild string
		wantErr   bool
	}{
		{"basic", "v1.2.3", 1, 2, 3, "", "", false},
		{"no v prefix", "1.2.3", 1, 2, 3, "", "", false},
		{"alpha prerelease", "v1.0.0-alpha.1", 1, 0, 0, "alpha.1", "", false},
		{"beta prerelease", "v2.1.0-beta.3", 2, 1, 0, "beta.3", "", false},
		{"rc prerelease", "v1.0.0-rc.1", 1, 0, 0, "rc.1", "", false},
		{"build metadata", "v1.0.0+build.123", 1, 0, 0, "", "build.123", false},
		{"pre + build", "v1.0.0-alpha.1+build.456", 1, 0, 0, "alpha.1", "build.456", false},
		{"zero minor patch", "v3.0.0", 3, 0, 0, "", "", false},
		{"large numbers", "v100.200.300", 100, 200, 300, "", "", false},
		{"invalid empty", "", 0, 0, 0, "", "", true},
		{"invalid word", "latest", 0, 0, 0, "", "", true},
		{"invalid partial", "v1.2", 0, 0, 0, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := ParseSemVer(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMajor, sv.Major)
			assert.Equal(t, tt.wantMinor, sv.Minor)
			assert.Equal(t, tt.wantPatch, sv.Patch)
			assert.Equal(t, tt.wantPre, sv.Prerelease)
			assert.Equal(t, tt.wantBuild, sv.Build)
		})
	}
}

func TestSemVerString(t *testing.T) {
	tests := []struct {
		sv   SemVer
		want string
	}{
		{SemVer{Major: 1, Minor: 2, Patch: 3}, "v1.2.3"},
		{SemVer{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha.1"}, "v1.0.0-alpha.1"},
		{SemVer{Major: 2, Minor: 0, Patch: 0, Build: "build.5"}, "v2.0.0+build.5"},
		{SemVer{Major: 1, Minor: 0, Patch: 0, Prerelease: "beta.1", Build: "sha.abc"}, "v1.0.0-beta.1+sha.abc"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.sv.String())
		})
	}
}

func TestIsPrerelease(t *testing.T) {
	sv, _ := ParseSemVer("v1.0.0-alpha.1")
	assert.True(t, sv.IsPrerelease())

	sv, _ = ParseSemVer("v1.0.0")
	assert.False(t, sv.IsPrerelease())
}

// ---------------------------------------------------------------------------
// CompareSemVer
// ---------------------------------------------------------------------------

func TestCompareSemVer(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal", "v1.0.0", "v1.0.0", 0},
		{"major diff", "v2.0.0", "v1.0.0", 1},
		{"minor diff", "v1.2.0", "v1.1.0", 1},
		{"patch diff", "v1.0.2", "v1.0.1", 1},
		{"prerelease < release", "v1.0.0-alpha.1", "v1.0.0", -1},
		{"release > prerelease", "v1.0.0", "v1.0.0-beta.1", 1},
		{"alpha < beta", "v1.0.0-alpha.1", "v1.0.0-beta.1", -1},
		{"alpha.1 < alpha.2", "v1.0.0-alpha.1", "v1.0.0-alpha.2", -1},
		{"beta.10 > beta.2", "v1.0.0-beta.10", "v1.0.0-beta.2", 1},
		{"equal prerelease", "v1.0.0-alpha.1", "v1.0.0-alpha.1", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := ParseSemVer(tt.a)
			b, _ := ParseSemVer(tt.b)
			assert.Equal(t, tt.want, CompareSemVer(a, b))
		})
	}
}

// ---------------------------------------------------------------------------
// LatestVersion
// ---------------------------------------------------------------------------

func TestLatestVersion(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		major    int
		want     string
	}{
		{
			name:     "picks highest",
			versions: []string{"v1.0.0", "v1.1.0", "v1.0.1"},
			major:    1,
			want:     "v1.1.0",
		},
		{
			name:     "filters by major",
			versions: []string{"v1.0.0", "v2.0.0", "v1.2.0"},
			major:    1,
			want:     "v1.2.0",
		},
		{
			name:     "prerelease lower than release",
			versions: []string{"v1.0.0-alpha.1", "v1.0.0"},
			major:    1,
			want:     "v1.0.0",
		},
		{
			name:     "all prerelease",
			versions: []string{"v1.0.0-alpha.1", "v1.0.0-beta.1"},
			major:    1,
			want:     "v1.0.0-beta.1",
		},
		{
			name:     "no matching major",
			versions: []string{"v2.0.0", "v3.0.0"},
			major:    1,
			want:     "",
		},
		{
			name:     "empty list",
			versions: nil,
			major:    1,
			want:     "",
		},
		{
			name:     "skips invalid",
			versions: []string{"v1.0.0", "latest", "v1.1.0"},
			major:    1,
			want:     "v1.1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LatestVersion(tt.versions, tt.major)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// LatestVersionFromTags
// ---------------------------------------------------------------------------

func TestLatestVersionFromTags(t *testing.T) {
	tags := []string{
		"proto/payments/ledger/v1/v1.0.0",
		"proto/payments/ledger/v1/v1.1.0",
		"proto/payments/ledger/v1/v1.0.1",
		"proto/billing/invoices/v1/v1.0.0",
	}
	got, err := LatestVersionFromTags(tags, "proto/payments/ledger/v1", 1)
	require.NoError(t, err)
	assert.Equal(t, "v1.1.0", got)

	// Non-matching prefix
	got, err = LatestVersionFromTags(tags, "proto/users/auth/v1", 1)
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

// ---------------------------------------------------------------------------
// SortVersions
// ---------------------------------------------------------------------------

func TestSortVersions(t *testing.T) {
	input := []string{"v1.2.0", "v1.0.0", "v1.1.0", "v1.0.0-alpha.1"}
	got := SortVersions(input)
	assert.Equal(t, []string{"v1.0.0-alpha.1", "v1.0.0", "v1.1.0", "v1.2.0"}, got)
}

// ---------------------------------------------------------------------------
// ValidateVersionLine
// ---------------------------------------------------------------------------

func TestValidateVersionLine(t *testing.T) {
	tests := []struct {
		name    string
		version string
		line    string
		wantErr string
	}{
		{"v1 match", "v1.2.3", "v1", ""},
		{"v2 match", "v2.0.0-beta.1", "v2", ""},
		{"v1 mismatch", "v2.0.0", "v1", "major 2 but API line"},
		{"v3 mismatch", "v1.0.0", "v3", "major 1 but API line"},
		{"invalid version", "latest", "v1", "invalid semver"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersionLine(tt.version, tt.line)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SuggestVersion
// ---------------------------------------------------------------------------

func TestSuggestVersion_InitialRelease(t *testing.T) {
	s, err := SuggestVersion("", false, true, "stable", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", s.Suggested)
	assert.Contains(t, s.Reasoning, "Initial release")
}

func TestSuggestVersion_InitialAlpha(t *testing.T) {
	s, err := SuggestVersion("", false, true, "experimental", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0-alpha.1", s.Suggested)
}

func TestSuggestVersion_InitialBeta(t *testing.T) {
	s, err := SuggestVersion("", false, true, "beta", "v2")
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0-beta.1", s.Suggested)
}

func TestSuggestVersion_MinorBump(t *testing.T) {
	s, err := SuggestVersion("v1.2.0", false, true, "stable", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1.3.0", s.Suggested)
	assert.Equal(t, BumpMinor, s.Bump)
}

func TestSuggestVersion_PatchBump(t *testing.T) {
	s, err := SuggestVersion("v1.2.3", false, false, "stable", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1.2.4", s.Suggested)
	assert.Equal(t, BumpPatch, s.Bump)
}

func TestSuggestVersion_MinorBumpWithBeta(t *testing.T) {
	s, err := SuggestVersion("v1.2.0", false, true, "beta", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1.3.0-beta.1", s.Suggested)
}

func TestSuggestVersion_BreakingRejected(t *testing.T) {
	s, err := SuggestVersion("v1.2.0", true, true, "stable", "v1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "breaking changes")
	assert.Contains(t, err.Error(), "v2")
	assert.NotNil(t, s)
	assert.Equal(t, BumpMajor, s.Bump)
}

func TestSuggestVersion_SunsetBlocked(t *testing.T) {
	_, err := SuggestVersion("v1.0.0", false, true, "sunset", "v1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sunset")
}

func TestSuggestVersion_PrereleaseBump(t *testing.T) {
	s, err := SuggestVersion("v1.0.0-alpha.1", false, true, "experimental", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0-alpha.2", s.Suggested)
}

func TestSuggestVersion_PrereleaseTransition(t *testing.T) {
	s, err := SuggestVersion("v1.0.0-alpha.3", false, true, "beta", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0-beta.1", s.Suggested)
	assert.Contains(t, s.Reasoning, "Transitioned")
}

func TestSuggestVersion_V2Line(t *testing.T) {
	s, err := SuggestVersion("", false, true, "stable", "v2")
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", s.Suggested)
}

func TestSuggestVersion_DeprecatedAllowed(t *testing.T) {
	s, err := SuggestVersion("v1.5.0", false, true, "deprecated", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1.6.0", s.Suggested)
}

// ---------------------------------------------------------------------------
// FormatSuggestionReport
// ---------------------------------------------------------------------------

func TestFormatSuggestionReport(t *testing.T) {
	s := &VersionSuggestion{
		Current:   "v1.0.0",
		Suggested: "v1.1.0",
		Bump:      BumpMinor,
		Reasoning: "Non-breaking additive changes detected",
	}
	report := FormatSuggestionReport(s)
	assert.Contains(t, report, "v1.0.0")
	assert.Contains(t, report, "v1.1.0")
	assert.Contains(t, report, "MINOR")
}

func TestFormatSuggestionReport_NoCurrentVersion(t *testing.T) {
	s := &VersionSuggestion{
		Current:   "",
		Suggested: "v1.0.0",
		Bump:      BumpMinor,
		Reasoning: "Initial release for line v1",
	}
	report := FormatSuggestionReport(s)
	assert.Contains(t, report, "first release")
}

func TestFormatSuggestionReport_Blocked(t *testing.T) {
	s := &VersionSuggestion{
		Current:   "v1.2.0",
		Suggested: "",
		Bump:      BumpMajor,
		Reasoning: "Breaking changes detected",
	}
	report := FormatSuggestionReport(s)
	assert.Contains(t, report, "blocked")
}

// ---------------------------------------------------------------------------
// lifecyclePrerelease / lifecyclePrereleasePrefix
// ---------------------------------------------------------------------------

func TestLifecyclePrerelease(t *testing.T) {
	assert.Equal(t, "alpha.1", lifecyclePrerelease("experimental", 1))
	assert.Equal(t, "alpha.3", lifecyclePrerelease("experimental", 3))
	assert.Equal(t, "beta.1", lifecyclePrerelease("preview", 1))
	assert.Equal(t, "beta.1", lifecyclePrerelease("beta", 1)) // canonical form
	assert.Equal(t, "", lifecyclePrerelease("stable", 1))
	assert.Equal(t, "", lifecyclePrerelease("deprecated", 1))
	assert.Equal(t, "", lifecyclePrerelease("", 1))
}

// ---------------------------------------------------------------------------
// v0 line support in SuggestVersion
// ---------------------------------------------------------------------------

func TestSuggestVersion_V0BreakingAllowed(t *testing.T) {
	// v0 allows breaking changes — they result in a minor bump
	s, err := SuggestVersion("v0.2.0", true, true, "experimental", "v0")
	require.NoError(t, err)
	assert.Equal(t, "v0.3.0-alpha.1", s.Suggested)
	assert.Equal(t, BumpMinor, s.Bump)
	assert.Contains(t, s.Reasoning, "v0")
}

func TestSuggestVersion_V0BreakingInitial(t *testing.T) {
	// v0 initial with breaking changes
	s, err := SuggestVersion("", true, true, "experimental", "v0")
	require.NoError(t, err)
	assert.Equal(t, "v0.1.0-alpha.1", s.Suggested)
	assert.Equal(t, BumpMinor, s.Bump)
}

func TestSuggestVersion_V0NonBreaking(t *testing.T) {
	// v0 non-breaking additive change → minor bump
	s, err := SuggestVersion("v0.1.0-alpha.1", false, true, "experimental", "v0")
	require.NoError(t, err)
	assert.Equal(t, "v0.1.0-alpha.2", s.Suggested) // prerelease bump
}

func TestSuggestVersion_V0Preview(t *testing.T) {
	s, err := SuggestVersion("", false, true, "preview", "v0")
	require.NoError(t, err)
	assert.Equal(t, "v0.0.0-beta.1", s.Suggested)
}

func TestValidateVersionLine_V0(t *testing.T) {
	assert.NoError(t, ValidateVersionLine("v0.1.0", "v0"))
	assert.NoError(t, ValidateVersionLine("v0.1.0-alpha.1", "v0"))
	assert.Error(t, ValidateVersionLine("v1.0.0", "v0"))
}
