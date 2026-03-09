package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJavaPlugin_Name(t *testing.T) {
	p := Get("java")
	require.NotNil(t, p)
	assert.Equal(t, "java", p.Name())
}

func TestJavaPlugin_RequiresOrg(t *testing.T) {
	p := Get("java")
	assert.False(t, p.Available(DerivationContext{Org: ""}))
	assert.True(t, p.Available(DerivationContext{Org: "acme"}))
}

func TestJavaPlugin_DeriveCoords(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	p := Get("java")
	coords, err := p.DeriveCoords(DerivationContext{Org: "acme", API: api})
	require.NoError(t, err)
	assert.Equal(t, "com.acme.apis:payments-ledger-v1-proto", coords.Module)
	assert.Equal(t, "com.acme.apis.payments.ledger.v1", coords.Import)
}

func TestJavaPlugin_ReportLines(t *testing.T) {
	p := Get("java")
	lines := p.ReportLines(config.LanguageCoords{Module: "coords", Import: "pkg"})
	require.Len(t, lines, 2)
	assert.Equal(t, "Maven", lines[0].Label)
	assert.Equal(t, "Java pkg", lines[1].Label)
}

func TestJavaPlugin_UnlinkHint(t *testing.T) {
	api, _ := config.ParseAPIID("proto/payments/ledger/v1")
	p := Get("java")
	hint := p.UnlinkHint(DerivationContext{Org: "acme", API: api})
	require.NotNil(t, hint)
	assert.Contains(t, hint.Message, "pom.xml")
}
