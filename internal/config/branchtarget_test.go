package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveTargetBranch(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		configured map[string]string
		want       string
	}{
		// Built-in defaults (no config).
		{"default main", "main", nil, "main"},
		{"default master to main", "master", nil, "main"},
		{"default develop", "develop", nil, "develop"},
		{"unknown branch falls back to main", "feature/x", nil, "main"},
		{"empty source falls back to main", "", nil, "main"},

		// Configured mapping overrides / extends the default (tweakable).
		{"configured develop to staging", "develop", map[string]string{"develop": "staging"}, "staging"},
		{"configured extra branch", "release-1.x", map[string]string{"release-1.x": "release-1.x"}, "release-1.x"},
		{"configured falls through to default for unlisted", "develop", map[string]string{"main": "main"}, "develop"},
		{"empty configured value falls through to default", "develop", map[string]string{"develop": ""}, "develop"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTargetBranch(tt.source, tt.configured)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPrereleaseChannel(t *testing.T) {
	assert.False(t, IsPrereleaseChannel("main"))
	assert.False(t, IsPrereleaseChannel(""))
	assert.True(t, IsPrereleaseChannel("develop"))
	assert.True(t, IsPrereleaseChannel("staging"))
}

func TestDefaultBranchTargets(t *testing.T) {
	d := DefaultBranchTargets()
	assert.Equal(t, "main", d["main"])
	assert.Equal(t, "main", d["master"])
	assert.Equal(t, "develop", d["develop"])
}
