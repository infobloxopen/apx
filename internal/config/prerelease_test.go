package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortCommitHash(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"1a2b3c4d5e6f7a8b9c0d", "1a2b3c4d5e6f"}, // truncated to 12
		{"1A2B3C4", "1a2b3c4"},                   // lowercased, short but valid
		{"  1a2b3c4d  ", "1a2b3c4d"},             // trimmed
		{"", ""},                                 // empty → none
		{"nothex!", ""},                          // non-hex → none
		{"123", ""},                              // too short (<7) → none
		{"1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9012", "1a2b3c4d5e6f"}, // 40-char full sha
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ShortCommitHash(tt.in))
		})
	}
}

// TestEncodeCommitHash covers AC-2: the hash goes in the pre-release segment,
// g-prefixed, valid SemVer, and Go-module resolvable.
func TestEncodeCommitHash(t *testing.T) {
	t.Run("appends g-prefixed hash to prerelease", func(t *testing.T) {
		got, err := EncodeCommitHash("v1.2.0-beta.1", "1a2b3c4d5e6f7a8b")
		require.NoError(t, err)
		assert.Equal(t, "v1.2.0-beta.1.g1a2b3c4d5e6f", got)

		// Result must parse as valid SemVer with no build metadata.
		sv, perr := ParseSemVer(got)
		require.NoError(t, perr)
		assert.Equal(t, "beta.1.g1a2b3c4d5e6f", sv.Prerelease)
		assert.Empty(t, sv.Build)
	})

	t.Run("idempotent", func(t *testing.T) {
		once, err := EncodeCommitHash("v1.2.0-beta.1", "1a2b3c4d5e6f")
		require.NoError(t, err)
		twice, err := EncodeCommitHash(once, "1a2b3c4d5e6f")
		require.NoError(t, err)
		assert.Equal(t, once, twice)
	})

	t.Run("drops build metadata", func(t *testing.T) {
		got, err := EncodeCommitHash("v1.2.0-beta.1+build.9", "abcdef1")
		require.NoError(t, err)
		assert.NotContains(t, got, "+")
		assert.Equal(t, "v1.2.0-beta.1.gabcdef1", got)
	})

	t.Run("no hash leaves version unchanged", func(t *testing.T) {
		got, err := EncodeCommitHash("v1.2.0-beta.1", "")
		require.NoError(t, err)
		assert.Equal(t, "v1.2.0-beta.1", got)
	})

	t.Run("rejects stable version", func(t *testing.T) {
		_, err := EncodeCommitHash("v1.2.0", "1a2b3c4d5e6f")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pre-release segment")
	})

	t.Run("rejects invalid version", func(t *testing.T) {
		_, err := EncodeCommitHash("not-a-version", "1a2b3c4d5e6f")
		require.Error(t, err)
	})
}

func TestHighestReleased(t *testing.T) {
	versions := []string{"v1.0.0", "v1.1.0", "v1.1.1", "v1.2.0-beta.1", "v2.0.0", "v0.9.0"}

	t.Run("v1 line ignores prereleases and other majors", func(t *testing.T) {
		got := HighestReleased(versions, 1)
		require.NotNil(t, got)
		assert.Equal(t, "v1.1.1", got.String())
	})

	t.Run("no GA yet returns nil", func(t *testing.T) {
		got := HighestReleased([]string{"v3.0.0-beta.1", "v3.0.0-beta.2"}, 3)
		assert.Nil(t, got)
	})

	t.Run("empty returns nil", func(t *testing.T) {
		assert.Nil(t, HighestReleased(nil, 1))
	})
}

func TestNextLegalPrerelease(t *testing.T) {
	ga, _ := ParseSemVer("v1.1.1")
	assert.Equal(t, "v1.1.2-beta.1", NextLegalPrerelease(ga))
	assert.Equal(t, "", NextLegalPrerelease(nil))
}

// TestAssertPrereleaseRatchet covers AC-1: fail closed when a pre-release is not
// strictly greater than the line's highest GA.
func TestAssertPrereleaseRatchet(t *testing.T) {
	released := []string{"v1.0.0", "v1.1.0", "v1.1.1"}

	t.Run("prerelease equal-base to GA is rejected", func(t *testing.T) {
		// v1.1.1-beta.* < v1.1.1 GA → must fail, suggesting v1.1.2-beta.1.
		err := AssertPrereleaseRatchet("v1.1.1-beta.1", released, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "v1.1.2-beta.1")
	})

	t.Run("prerelease below GA is rejected", func(t *testing.T) {
		err := AssertPrereleaseRatchet("v1.0.5-beta.1", released, 1)
		require.Error(t, err)
	})

	t.Run("prerelease above GA is allowed", func(t *testing.T) {
		err := AssertPrereleaseRatchet("v1.1.2-beta.1", released, 1)
		require.NoError(t, err)
	})

	t.Run("prerelease with hash above GA is allowed", func(t *testing.T) {
		err := AssertPrereleaseRatchet("v1.2.0-beta.1.g1a2b3c4d5e6f", released, 1)
		require.NoError(t, err)
	})

	t.Run("stable version is never ratcheted", func(t *testing.T) {
		assert.NoError(t, AssertPrereleaseRatchet("v1.1.1", released, 1))
		assert.NoError(t, AssertPrereleaseRatchet("v1.0.0", released, 1))
	})

	t.Run("no GA on line allows any prerelease", func(t *testing.T) {
		assert.NoError(t, AssertPrereleaseRatchet("v2.0.0-beta.1", released, 2))
	})
}
