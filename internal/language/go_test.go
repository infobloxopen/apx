package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoPlugin_Name(t *testing.T) {
	p := Get("go")
	require.NotNil(t, p)
	assert.Equal(t, "go", p.Name())
}

func TestGoPlugin_Tier(t *testing.T) {
	p := Get("go")
	assert.Equal(t, 1, p.Tier())
}

func TestGoPlugin_AlwaysAvailable(t *testing.T) {
	p := Get("go")
	assert.True(t, p.Available(DerivationContext{Org: ""}))
	assert.True(t, p.Available(DerivationContext{Org: "acme"}))
}

func TestGoPlugin_DeriveCoords(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	p := Get("go")
	coords, err := p.DeriveCoords(DerivationContext{
		SourceRepo: "github.com/acme/apis",
		API:        api,
	})
	require.NoError(t, err)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", coords.Module)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v1", coords.Import)
}

func TestGoPlugin_ReportLines(t *testing.T) {
	p := Get("go")
	lines := p.ReportLines(config.LanguageCoords{Module: "mod", Import: "imp"})
	require.Len(t, lines, 2)
	assert.Equal(t, "Go module", lines[0].Label)
	assert.Equal(t, "Go import", lines[1].Label)
}

func TestGoPlugin_UnlinkHint(t *testing.T) {
	api, _ := config.ParseAPIID("proto/payments/ledger/v1")
	p := Get("go")
	hint := p.UnlinkHint(DerivationContext{
		SourceRepo: "github.com/acme/apis",
		API:        api,
	})
	require.NotNil(t, hint)
	assert.Contains(t, hint.Message, "go get")
}

func TestGoPlugin_ImplementsPostGenHook(t *testing.T) {
	p := Get("go")
	_, ok := p.(PostGenHook)
	assert.True(t, ok, "Go plugin should implement PostGenHook")
}

// ---------------------------------------------------------------------------
// Go identity derivation tests (moved from config/identity_test.go)
// ---------------------------------------------------------------------------

func TestDeriveGoModule(t *testing.T) {
	tests := []struct {
		name       string
		sourceRepo string
		api        *config.APIIdentity
		want       string
	}{
		{
			name:       "v1 module has no version suffix",
			sourceRepo: "github.com/acme/apis",
			api:        &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want:       "github.com/acme/apis/proto/payments/ledger",
		},
		{
			name:       "v2 module has version suffix",
			sourceRepo: "github.com/acme/apis",
			api:        &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v2"},
			want:       "github.com/acme/apis/proto/payments/ledger/v2",
		},
		{
			name:       "v3 module has version suffix",
			sourceRepo: "github.com/acme/apis",
			api:        &config.APIIdentity{Format: "openapi", Domain: "billing", Name: "invoices", Line: "v3"},
			want:       "github.com/acme/apis/openapi/billing/invoices/v3",
		},
		{
			name:       "v0 module has no version suffix",
			sourceRepo: "github.com/acme/apis",
			api:        &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v0"},
			want:       "github.com/acme/apis/proto/payments/ledger",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deriveGoModule(tt.sourceRepo, tt.api)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeriveGoImport(t *testing.T) {
	tests := []struct {
		name       string
		sourceRepo string
		api        *config.APIIdentity
		want       string
	}{
		{
			name:       "v1 import includes v1",
			sourceRepo: "github.com/acme/apis",
			api:        &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want:       "github.com/acme/apis/proto/payments/ledger/v1",
		},
		{
			name:       "v2 import includes v2",
			sourceRepo: "github.com/acme/apis",
			api:        &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v2"},
			want:       "github.com/acme/apis/proto/payments/ledger/v2",
		},
		{
			name:       "v0 import includes v0",
			sourceRepo: "github.com/acme/apis",
			api:        &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v0"},
			want:       "github.com/acme/apis/proto/payments/ledger/v0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deriveGoImport(tt.sourceRepo, tt.api)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
