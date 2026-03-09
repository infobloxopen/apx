package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubPlugin is a minimal LanguagePlugin implementation for testing.
type stubPlugin struct {
	name  string
	tier  int
	avail bool
}

func (s *stubPlugin) Name() string                         { return s.name }
func (s *stubPlugin) Tier() int                            { return s.tier }
func (s *stubPlugin) Available(ctx DerivationContext) bool { return s.avail }
func (s *stubPlugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error) {
	return config.LanguageCoords{Module: s.name + "-mod", Import: s.name + "-imp"}, nil
}
func (s *stubPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	return []ReportLine{{Label: s.name, Value: coords.Module}}
}
func (s *stubPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	return &UnlinkHint{Message: s.name + " unlink hint"}
}

func TestRegister(t *testing.T) {
	restore := resetForTesting()
	defer restore()

	p := &stubPlugin{name: "test-lang", tier: 2, avail: true}
	Register(p)

	got := Get("test-lang")
	require.NotNil(t, got)
	assert.Equal(t, "test-lang", got.Name())
}

func TestRegisterDuplicatePanics(t *testing.T) {
	restore := resetForTesting()
	defer restore()

	p := &stubPlugin{name: "dup", tier: 2, avail: true}
	Register(p)

	assert.Panics(t, func() {
		Register(&stubPlugin{name: "dup", tier: 2, avail: true})
	})
}

func TestGetReturnsNilForUnknown(t *testing.T) {
	restore := resetForTesting()
	defer restore()

	assert.Nil(t, Get("nonexistent"))
}

func TestAllSortedByTierThenName(t *testing.T) {
	restore := resetForTesting()
	defer restore()

	Register(&stubPlugin{name: "typescript", tier: 2, avail: true})
	Register(&stubPlugin{name: "go", tier: 1, avail: true})
	Register(&stubPlugin{name: "python", tier: 2, avail: true})
	Register(&stubPlugin{name: "java", tier: 2, avail: true})

	all := All()
	require.Len(t, all, 4)
	assert.Equal(t, "go", all[0].Name())         // Tier 1 first
	assert.Equal(t, "java", all[1].Name())       // Tier 2, alpha
	assert.Equal(t, "python", all[2].Name())     // Tier 2, alpha
	assert.Equal(t, "typescript", all[3].Name()) // Tier 2, alpha
}

func TestAvailableFilters(t *testing.T) {
	restore := resetForTesting()
	defer restore()

	Register(&stubPlugin{name: "go", tier: 1, avail: true})
	Register(&stubPlugin{name: "python", tier: 2, avail: false})
	Register(&stubPlugin{name: "java", tier: 2, avail: true})

	ctx := DerivationContext{}
	avail := Available(ctx)
	require.Len(t, avail, 2)
	assert.Equal(t, "go", avail[0].Name())
	assert.Equal(t, "java", avail[1].Name())
}

func TestNames(t *testing.T) {
	restore := resetForTesting()
	defer restore()

	Register(&stubPlugin{name: "go", tier: 1, avail: true})
	Register(&stubPlugin{name: "python", tier: 2, avail: true})

	names := Names()
	assert.Equal(t, []string{"go", "python"}, names)
}

// TestRealPluginsRegistered verifies that init() registration
// actually works — all 6 built-in plugins should be present.
func TestRealPluginsRegistered(t *testing.T) {
	all := All()
	require.Len(t, all, 6)
	assert.Equal(t, "go", all[0].Name())         // Tier 1
	assert.Equal(t, "cpp", all[1].Name())        // Tier 2, alpha
	assert.Equal(t, "java", all[2].Name())       // Tier 2, alpha
	assert.Equal(t, "python", all[3].Name())     // Tier 2, alpha
	assert.Equal(t, "rust", all[4].Name())       // Tier 2, alpha
	assert.Equal(t, "typescript", all[5].Name()) // Tier 2, alpha
}
