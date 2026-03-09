package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypescriptPlugin_Name(t *testing.T) {
	p := Get("typescript")
	require.NotNil(t, p)
	assert.Equal(t, "typescript", p.Name())
}

func TestTypescriptPlugin_RequiresOrg(t *testing.T) {
	p := Get("typescript")
	assert.False(t, p.Available(DerivationContext{Org: ""}))
	assert.True(t, p.Available(DerivationContext{Org: "acme"}))
}

func TestTypescriptPlugin_DeriveCoords(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	p := Get("typescript")
	coords, err := p.DeriveCoords(DerivationContext{Org: "acme", API: api})
	require.NoError(t, err)
	assert.Equal(t, "@acme/payments-ledger-v1-proto", coords.Module)
	assert.Equal(t, "@acme/payments-ledger-v1-proto", coords.Import)
}

func TestTypescriptPlugin_ReportLines(t *testing.T) {
	p := Get("typescript")
	lines := p.ReportLines(config.LanguageCoords{Module: "@acme/pkg", Import: "@acme/pkg"})
	require.Len(t, lines, 1, "TypeScript should have one line (module == import)")
	assert.Equal(t, "npm", lines[0].Label)
}

func TestTypescriptPlugin_UnlinkHint(t *testing.T) {
	api, _ := config.ParseAPIID("proto/payments/ledger/v1")
	p := Get("typescript")
	hint := p.UnlinkHint(DerivationContext{Org: "acme", API: api})
	require.NotNil(t, hint)
	assert.Contains(t, hint.Message, "npm install")
}
