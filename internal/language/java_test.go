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

// ---------------------------------------------------------------------------
// Java / Maven identity derivation tests (moved from config/identity_test.go)
// ---------------------------------------------------------------------------

func TestDeriveMavenGroupId(t *testing.T) {
	tests := []struct {
		name string
		org  string
		want string
	}{
		{"simple org", "acme", "com.acme.apis"},
		{"uppercase org", "ACME", "com.acme.apis"},
		{"hyphenated org", "acme-corp", "com.acme.corp.apis"},
		{"mixed case hyphenated", "Acme-Corp", "com.acme.corp.apis"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveMavenGroupId(tt.org))
		})
	}
}

func TestDeriveMavenArtifactId(t *testing.T) {
	tests := []struct {
		name string
		api  *config.APIIdentity
		want string
	}{
		{
			name: "4-part with domain",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "payments-ledger-v1-proto",
		},
		{
			name: "3-part no domain",
			api:  &config.APIIdentity{Format: "proto", Domain: "", Name: "orders", Line: "v1"},
			want: "orders-v1-proto",
		},
		{
			name: "v0 line",
			api:  &config.APIIdentity{Format: "proto", Domain: "events", Name: "click", Line: "v0"},
			want: "events-click-v0-proto",
		},
		{
			name: "v2 line",
			api:  &config.APIIdentity{Format: "proto", Domain: "billing", Name: "invoices", Line: "v2"},
			want: "billing-invoices-v2-proto",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveMavenArtifactId(tt.api))
		})
	}
}

func TestDeriveMavenCoords(t *testing.T) {
	api := &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"}
	got := deriveMavenCoords("acme", api)
	assert.Equal(t, "com.acme.apis:payments-ledger-v1-proto", got)
}

func TestDeriveJavaPackage(t *testing.T) {
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
			want: "com.acme.apis.payments.ledger.v1",
		},
		{
			name: "3-part no domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "", Name: "orders", Line: "v1"},
			want: "com.acme.apis.orders.v1",
		},
		{
			name: "hyphenated org",
			org:  "acme-corp",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "com.acme.corp.apis.payments.ledger.v1",
		},
		{
			name: "uppercase org normalized",
			org:  "ACME",
			api:  &config.APIIdentity{Format: "proto", Domain: "billing", Name: "invoices", Line: "v2"},
			want: "com.acme.apis.billing.invoices.v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveJavaPackage(tt.org, tt.api))
		})
	}
}
