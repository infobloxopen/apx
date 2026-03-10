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

// ---------------------------------------------------------------------------
// Python identity derivation tests (moved from config/identity_test.go)
// ---------------------------------------------------------------------------

func TestDerivePythonDistName(t *testing.T) {
	tests := []struct {
		name string
		org  string
		api  *config.APIIdentity
		want string
	}{
		{
			name: "4-part with domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "acme-payments-ledger-v1",
		},
		{
			name: "3-part no domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "", Name: "orders", Line: "v1"},
			want: "acme-orders-v1",
		},
		{
			name: "uppercase org normalized",
			org:  "ACME",
			api:  &config.APIIdentity{Format: "proto", Domain: "Payments", Name: "Ledger", Line: "v2"},
			want: "acme-payments-ledger-v2",
		},
		{
			name: "v0 line",
			org:  "myorg",
			api:  &config.APIIdentity{Format: "avro", Domain: "events", Name: "click", Line: "v0"},
			want: "myorg-events-click-v0",
		},
		{
			name: "hyphenated org kept in dist name",
			org:  "acme-corp",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "acme-corp-payments-ledger-v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, derivePythonDistName(tt.org, tt.api))
		})
	}
}

func TestDerivePythonImport(t *testing.T) {
	tests := []struct {
		name string
		org  string
		api  *config.APIIdentity
		want string
	}{
		{
			name: "4-part with domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "acme_apis.payments.ledger.v1",
		},
		{
			name: "3-part no domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "", Name: "orders", Line: "v1"},
			want: "acme_apis.orders.v1",
		},
		{
			name: "uppercase org normalized",
			org:  "ACME",
			api:  &config.APIIdentity{Format: "proto", Domain: "Payments", Name: "Ledger", Line: "v2"},
			want: "acme_apis.payments.ledger.v2",
		},
		{
			name: "v0 line",
			org:  "myorg",
			api:  &config.APIIdentity{Format: "avro", Domain: "events", Name: "click", Line: "v0"},
			want: "myorg_apis.events.click.v0",
		},
		{
			name: "hyphenated org normalized to underscores",
			org:  "acme-corp",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "acme_corp_apis.payments.ledger.v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, derivePythonImport(tt.org, tt.api))
		})
	}
}

func TestNormalizePEP440Version(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"v1.0.0-beta.1", "1.0.0b1"},
		{"v1.0.0-alpha.2", "1.0.0a2"},
		{"v1.0.0-rc.1", "1.0.0rc1"},
		{"v2.1.0-beta.3", "2.1.0b3"},
		{"v0.1.0-alpha.1", "0.1.0a1"},
		{"v1.0.0", "1.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizePEP440Version(tt.input))
		})
	}
}
