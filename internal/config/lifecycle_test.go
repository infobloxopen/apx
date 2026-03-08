package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLifecycle(t *testing.T) {
	assert.Equal(t, "beta", NormalizeLifecycle("beta"))
	assert.Equal(t, "beta", NormalizeLifecycle("preview"))
	assert.Equal(t, "experimental", NormalizeLifecycle("experimental"))
	assert.Equal(t, "stable", NormalizeLifecycle("stable"))
	assert.Equal(t, "deprecated", NormalizeLifecycle("deprecated"))
	assert.Equal(t, "sunset", NormalizeLifecycle("sunset"))
}

func TestValidateVersionLifecycle_Experimental(t *testing.T) {
	// experimental requires -alpha.*
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-alpha.1", "experimental"))
	assert.NoError(t, ValidateVersionLifecycle("v0.1.0-alpha.3", "experimental"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0-beta.1", "experimental"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0", "experimental"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0-rc.1", "experimental"))
}

func TestValidateVersionLifecycle_Preview(t *testing.T) {
	// "preview" is an alias for "beta" — accepts alpha, beta, and rc prereleases
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-alpha.1", "preview"))
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-beta.1", "preview"))
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-rc.1", "preview"))
	assert.NoError(t, ValidateVersionLifecycle("v2.0.0-beta.5", "preview"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0", "preview"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0-dev.1", "preview"))
}

func TestValidateVersionLifecycle_Beta(t *testing.T) {
	// "beta" is the canonical lifecycle — accepts alpha, beta, and rc prereleases
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-beta.1", "beta"))
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-alpha.1", "beta"))
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-rc.1", "beta"))
	assert.NoError(t, ValidateVersionLifecycle("v2.0.0-beta.5", "beta"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0", "beta"))
}

func TestValidateVersionLifecycle_Stable(t *testing.T) {
	// stable must NOT have prerelease
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0", "stable"))
	assert.NoError(t, ValidateVersionLifecycle("v2.3.4", "stable"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0-beta.1", "stable"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0-alpha.1", "stable"))
}

func TestValidateVersionLifecycle_Deprecated(t *testing.T) {
	// deprecated allows anything
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0", "deprecated"))
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-beta.1", "deprecated"))
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-alpha.1", "deprecated"))
}

func TestValidateVersionLifecycle_Sunset(t *testing.T) {
	// sunset blocks everything
	assert.Error(t, ValidateVersionLifecycle("v1.0.0", "sunset"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0-beta.1", "sunset"))
}

func TestValidateVersionLifecycle_EmptyLifecycle(t *testing.T) {
	// Empty lifecycle skips validation
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-anything", ""))
}

func TestExtractPrerelease(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"v1.0.0", ""},
		{"v1.0.0-alpha.1", "alpha.1"},
		{"v1.0.0-beta.1", "beta.1"},
		{"v1.0.0-rc.1", "rc.1"},
		{"v1.0.0-alpha.1+build.123", "alpha.1"},
		{"1.0.0-beta.2", "beta.2"},
		{"v0.1.0-alpha.1", "alpha.1"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, extractPrerelease(tt.version))
		})
	}
}

func TestLifecycleAllowsRelease(t *testing.T) {
	assert.True(t, LifecycleAllowsRelease("experimental"))
	assert.True(t, LifecycleAllowsRelease("preview"))
	assert.True(t, LifecycleAllowsRelease("beta"))
	assert.True(t, LifecycleAllowsRelease("stable"))
	assert.True(t, LifecycleAllowsRelease("deprecated"))
	assert.False(t, LifecycleAllowsRelease("sunset"))
}

func TestLifecycleRequiresWarning(t *testing.T) {
	assert.True(t, LifecycleRequiresWarning("deprecated"))
	assert.False(t, LifecycleRequiresWarning("stable"))
	assert.False(t, LifecycleRequiresWarning("preview"))
	assert.False(t, LifecycleRequiresWarning("beta"))
	assert.False(t, LifecycleRequiresWarning("sunset"))
}

// ---------------------------------------------------------------------------
// ValidateLifecycleTransition
// ---------------------------------------------------------------------------

func TestValidateLifecycleTransition_Legal(t *testing.T) {
	// Forward transitions are legal
	assert.NoError(t, ValidateLifecycleTransition("experimental", "preview"))
	assert.NoError(t, ValidateLifecycleTransition("experimental", "beta")) // beta alias
	assert.NoError(t, ValidateLifecycleTransition("preview", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("beta", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("stable", "deprecated"))
	assert.NoError(t, ValidateLifecycleTransition("deprecated", "sunset"))
	// Skip transitions are also legal
	assert.NoError(t, ValidateLifecycleTransition("experimental", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("preview", "deprecated"))
	assert.NoError(t, ValidateLifecycleTransition("experimental", "sunset"))
}

func TestValidateLifecycleTransition_SameState(t *testing.T) {
	// Staying at the same state is always legal
	assert.NoError(t, ValidateLifecycleTransition("experimental", "experimental"))
	assert.NoError(t, ValidateLifecycleTransition("preview", "preview"))
	assert.NoError(t, ValidateLifecycleTransition("beta", "beta"))
	assert.NoError(t, ValidateLifecycleTransition("beta", "preview")) // same rank
	assert.NoError(t, ValidateLifecycleTransition("preview", "beta")) // same rank
	assert.NoError(t, ValidateLifecycleTransition("stable", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("deprecated", "deprecated"))
	assert.NoError(t, ValidateLifecycleTransition("sunset", "sunset"))
}

func TestValidateLifecycleTransition_Illegal(t *testing.T) {
	// Backward transitions are illegal
	assert.Error(t, ValidateLifecycleTransition("stable", "experimental"))
	assert.Error(t, ValidateLifecycleTransition("stable", "preview"))
	assert.Error(t, ValidateLifecycleTransition("stable", "beta"))
	assert.Error(t, ValidateLifecycleTransition("preview", "experimental"))
	assert.Error(t, ValidateLifecycleTransition("deprecated", "stable"))
	assert.Error(t, ValidateLifecycleTransition("sunset", "deprecated"))
	assert.Error(t, ValidateLifecycleTransition("sunset", "experimental"))
}

func TestValidateLifecycleTransition_EmptyFrom(t *testing.T) {
	// Empty "from" means fresh API — any target is legal
	assert.NoError(t, ValidateLifecycleTransition("", "experimental"))
	assert.NoError(t, ValidateLifecycleTransition("", "preview"))
	assert.NoError(t, ValidateLifecycleTransition("", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("", "sunset"))
}

func TestValidateLifecycleTransition_UnknownState(t *testing.T) {
	assert.Error(t, ValidateLifecycleTransition("unknown", "preview"))
	assert.Error(t, ValidateLifecycleTransition("preview", "unknown"))
}

// ---------------------------------------------------------------------------
// v0 line policy
// ---------------------------------------------------------------------------

func TestValidateV0Lifecycle(t *testing.T) {
	// v0 only allows experimental or beta (preview alias)
	assert.NoError(t, ValidateV0Lifecycle("experimental"))
	assert.NoError(t, ValidateV0Lifecycle("beta"))
	assert.NoError(t, ValidateV0Lifecycle("preview")) // alias for beta
	assert.Error(t, ValidateV0Lifecycle("stable"))
	assert.Error(t, ValidateV0Lifecycle("deprecated"))
	assert.Error(t, ValidateV0Lifecycle("sunset"))
}

func TestV0AllowsBreaking(t *testing.T) {
	assert.True(t, V0AllowsBreaking())
}

// ---------------------------------------------------------------------------
// Compatibility promise
// ---------------------------------------------------------------------------

func TestDeriveCompatibilityPromise(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		lifecycle string
		level     string
	}{
		{"v0 experimental", "v0", "experimental", "none"},
		{"v0 preview", "v0", "preview", "none"},
		{"v1 experimental", "v1", "experimental", "none"},
		{"v1 preview", "v1", "preview", "stabilizing"},
		{"v1 beta alias", "v1", "beta", "stabilizing"},
		{"v1 stable", "v1", "stable", "full"},
		{"v2 stable", "v2", "stable", "full"},
		{"v1 deprecated", "v1", "deprecated", "maintenance"},
		{"v1 sunset", "v1", "sunset", "eol"},
		{"unknown lifecycle", "v1", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promise := DeriveCompatibilityPromise(tt.line, tt.lifecycle)
			assert.Equal(t, tt.level, promise.Level)
			assert.NotEmpty(t, promise.Summary)
			assert.NotEmpty(t, promise.BreakingPolicy)
		})
	}
}

// ---------------------------------------------------------------------------
// Production recommendation
// ---------------------------------------------------------------------------

func TestProductionRecommendation(t *testing.T) {
	assert.Contains(t, ProductionRecommendation("experimental"), "Not recommended")
	assert.Contains(t, ProductionRecommendation("beta"), "caution")
	assert.Contains(t, ProductionRecommendation("preview"), "caution") // alias
	assert.Contains(t, ProductionRecommendation("stable"), "Recommended")
	assert.Contains(t, ProductionRecommendation("deprecated"), "Migrate")
	assert.Contains(t, ProductionRecommendation("sunset"), "Do not use")
	assert.Contains(t, ProductionRecommendation(""), "Unknown")
}
