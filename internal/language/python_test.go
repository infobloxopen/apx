package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonPlugin_Name(t *testing.T) {
	p := Get("python")
	require.NotNil(t, p)
	assert.Equal(t, "python", p.Name())
}

func TestPythonPlugin_RequiresOrg(t *testing.T) {
	p := Get("python")
	assert.False(t, p.Available(DerivationContext{Org: ""}))
	assert.True(t, p.Available(DerivationContext{Org: "acme"}))
}

func TestPythonPlugin_DeriveCoords(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	p := Get("python")
	coords, err := p.DeriveCoords(DerivationContext{Org: "acme", API: api})
	require.NoError(t, err)
	assert.Equal(t, "acme-payments-ledger-v1", coords.Module)
	assert.Equal(t, "acme_apis.payments.ledger.v1", coords.Import)
}

func TestPythonPlugin_ReportLines(t *testing.T) {
	p := Get("python")
	lines := p.ReportLines(config.LanguageCoords{Module: "dist", Import: "imp"})
	require.Len(t, lines, 2)
	assert.Equal(t, "Py dist", lines[0].Label)
	assert.Equal(t, "Py import", lines[1].Label)
}

func TestPythonPlugin_UnlinkHint(t *testing.T) {
	api, _ := config.ParseAPIID("proto/payments/ledger/v1")
	p := Get("python")
	hint := p.UnlinkHint(DerivationContext{Org: "acme", API: api})
	require.NotNil(t, hint)
	assert.Contains(t, hint.Message, "pip install")
}

func TestPythonPlugin_ImplementsScaffolder(t *testing.T) {
	p := Get("python")
	_, ok := p.(Scaffolder)
	assert.True(t, ok, "Python plugin should implement Scaffolder")
}

func TestPythonPlugin_ImplementsLinker(t *testing.T) {
	p := Get("python")
	_, ok := p.(Linker)
	assert.True(t, ok, "Python plugin should implement Linker")
}
