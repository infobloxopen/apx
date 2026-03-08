package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateVersionLifecycle_Experimental(t *testing.T) {
	// experimental requires -alpha.*
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-alpha.1", "experimental"))
	assert.NoError(t, ValidateVersionLifecycle("v0.1.0-alpha.3", "experimental"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0-beta.1", "experimental"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0", "experimental"))
}

func TestValidateVersionLifecycle_Beta(t *testing.T) {
	// beta requires -beta.*
	assert.NoError(t, ValidateVersionLifecycle("v1.0.0-beta.1", "beta"))
	assert.NoError(t, ValidateVersionLifecycle("v2.0.0-beta.5", "beta"))
	assert.Error(t, ValidateVersionLifecycle("v1.0.0-alpha.1", "beta"))
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
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, extractPrerelease(tt.version))
		})
	}
}

func TestLifecycleAllowsRelease(t *testing.T) {
	assert.True(t, LifecycleAllowsRelease("experimental"))
	assert.True(t, LifecycleAllowsRelease("beta"))
	assert.True(t, LifecycleAllowsRelease("stable"))
	assert.True(t, LifecycleAllowsRelease("deprecated"))
	assert.False(t, LifecycleAllowsRelease("sunset"))
}

func TestLifecycleRequiresWarning(t *testing.T) {
	assert.True(t, LifecycleRequiresWarning("deprecated"))
	assert.False(t, LifecycleRequiresWarning("stable"))
	assert.False(t, LifecycleRequiresWarning("beta"))
	assert.False(t, LifecycleRequiresWarning("sunset"))
}

// ---------------------------------------------------------------------------
// ValidateLifecycleTransition
// ---------------------------------------------------------------------------

func TestValidateLifecycleTransition_Legal(t *testing.T) {
	// Forward transitions are legal
	assert.NoError(t, ValidateLifecycleTransition("experimental", "beta"))
	assert.NoError(t, ValidateLifecycleTransition("beta", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("stable", "deprecated"))
	assert.NoError(t, ValidateLifecycleTransition("deprecated", "sunset"))
	// Skip transitions are also legal
	assert.NoError(t, ValidateLifecycleTransition("experimental", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("beta", "deprecated"))
	assert.NoError(t, ValidateLifecycleTransition("experimental", "sunset"))
}

func TestValidateLifecycleTransition_SameState(t *testing.T) {
	// Staying at the same state is always legal
	assert.NoError(t, ValidateLifecycleTransition("experimental", "experimental"))
	assert.NoError(t, ValidateLifecycleTransition("beta", "beta"))
	assert.NoError(t, ValidateLifecycleTransition("stable", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("deprecated", "deprecated"))
	assert.NoError(t, ValidateLifecycleTransition("sunset", "sunset"))
}

func TestValidateLifecycleTransition_Illegal(t *testing.T) {
	// Backward transitions are illegal
	assert.Error(t, ValidateLifecycleTransition("stable", "experimental"))
	assert.Error(t, ValidateLifecycleTransition("stable", "beta"))
	assert.Error(t, ValidateLifecycleTransition("beta", "experimental"))
	assert.Error(t, ValidateLifecycleTransition("deprecated", "stable"))
	assert.Error(t, ValidateLifecycleTransition("sunset", "deprecated"))
	assert.Error(t, ValidateLifecycleTransition("sunset", "experimental"))
}

func TestValidateLifecycleTransition_EmptyFrom(t *testing.T) {
	// Empty "from" means fresh API — any target is legal
	assert.NoError(t, ValidateLifecycleTransition("", "experimental"))
	assert.NoError(t, ValidateLifecycleTransition("", "stable"))
	assert.NoError(t, ValidateLifecycleTransition("", "sunset"))
}

func TestValidateLifecycleTransition_UnknownState(t *testing.T) {
	assert.Error(t, ValidateLifecycleTransition("unknown", "beta"))
	assert.Error(t, ValidateLifecycleTransition("beta", "unknown"))
}
