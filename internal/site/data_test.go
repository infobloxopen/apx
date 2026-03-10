package site

import (
	"testing"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Import language package to trigger all plugin init() registrations.
	_ "github.com/infobloxopen/apx/internal/language"
)

func TestBuildSiteData_EmptyCatalog(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []catalog.Module{},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "", "acme")

	assert.Equal(t, "acme", data.Org)
	assert.Equal(t, "apis", data.Repo)
	assert.Empty(t, data.APIs)
	assert.NotEmpty(t, data.GeneratedAt)
}

func TestBuildSiteData_SingleModule(t *testing.T) {
	cat := &catalog.Catalog{
		Version:    1,
		Org:        "acme",
		Repo:       "apis",
		ImportRoot: "go.acme.dev/apis",
		Modules: []catalog.Module{
			{
				ID:           "proto/payments/ledger/v1",
				Format:       "proto",
				Domain:       "payments",
				APILine:      "v1",
				Description:  "Payments ledger service API",
				Version:      "v1.2.3",
				LatestStable: "v1.2.3",
				Lifecycle:    "stable",
				Tags:         []string{"payments", "internal"},
				Owners:       []string{"payments-team"},
			},
		},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "go.acme.dev/apis", "acme")

	require.Len(t, data.APIs, 1)
	api := data.APIs[0]

	assert.Equal(t, "proto/payments/ledger/v1", api.ID)
	assert.Equal(t, "proto", api.Format)
	assert.Equal(t, "payments", api.Domain)
	assert.Equal(t, "ledger", api.Name)
	assert.Equal(t, "v1", api.Line)
	assert.Equal(t, "Payments ledger service API", api.Description)
	assert.Equal(t, "v1.2.3", api.LatestStable)
	assert.Equal(t, "stable", api.Lifecycle)
	assert.Equal(t, []string{"payments", "internal"}, api.Tags)
	assert.Equal(t, []string{"payments-team"}, api.Owners)
}

func TestBuildSiteData_LifecycleEnrichment(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []catalog.Module{
			{
				ID:        "proto/billing/invoices/v1",
				Format:    "proto",
				Domain:    "billing",
				APILine:   "v1",
				Lifecycle: "stable",
			},
		},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "", "acme")

	require.Len(t, data.APIs, 1)
	api := data.APIs[0]

	require.NotNil(t, api.Compatibility)
	assert.Equal(t, "full", api.Compatibility.Level)
	assert.Contains(t, api.Compatibility.Summary, "backward compatibility")
	assert.Contains(t, api.Compatibility.BreakingPolicy, "new major API line")
	assert.Contains(t, api.Compatibility.ProductionUse, "Recommended")
}

func TestBuildSiteData_LifecycleNormalization(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []catalog.Module{
			{
				ID:        "proto/orders/v1",
				Format:    "proto",
				APILine:   "v1",
				Lifecycle: "preview", // should normalize to "beta"
			},
		},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "", "acme")

	require.Len(t, data.APIs, 1)
	assert.Equal(t, "beta", data.APIs[0].Lifecycle)
}

func TestBuildSiteData_GoLanguageCoords(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []catalog.Module{
			{
				ID:      "proto/payments/ledger/v1",
				Format:  "proto",
				Domain:  "payments",
				APILine: "v1",
			},
		},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "", "acme")

	require.Len(t, data.APIs, 1)
	api := data.APIs[0]

	// Go is always available (Tier 1).
	goCoords, ok := api.Languages["go"]
	require.True(t, ok, "Go coordinates should always be present")
	require.Len(t, goCoords, 2)
	assert.Equal(t, "Go module", goCoords[0].Label)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", goCoords[0].Value)
	assert.Equal(t, "Go import", goCoords[1].Label)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v1", goCoords[1].Value)
}

func TestBuildSiteData_ImportRootAffectsGoCoords(t *testing.T) {
	cat := &catalog.Catalog{
		Version:    1,
		Org:        "acme",
		Repo:       "apis",
		ImportRoot: "go.acme.dev/apis",
		Modules: []catalog.Module{
			{
				ID:      "proto/payments/ledger/v1",
				Format:  "proto",
				Domain:  "payments",
				APILine: "v1",
			},
		},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "go.acme.dev/apis", "acme")

	require.Len(t, data.APIs, 1)
	goCoords := data.APIs[0].Languages["go"]
	require.Len(t, goCoords, 2)
	assert.Equal(t, "go.acme.dev/apis/proto/payments/ledger", goCoords[0].Value)
	assert.Equal(t, "go.acme.dev/apis/proto/payments/ledger/v1", goCoords[1].Value)
}

func TestBuildSiteData_MultiLanguageCoords(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []catalog.Module{
			{
				ID:      "proto/payments/ledger/v1",
				Format:  "proto",
				Domain:  "payments",
				APILine: "v1",
			},
		},
	}

	// With org set, all language plugins should be available.
	data := BuildSiteData(cat, "github.com/acme/apis", "", "acme")

	require.Len(t, data.APIs, 1)
	langs := data.APIs[0].Languages

	// Go is Tier 1 (always available).
	assert.Contains(t, langs, "go")

	// With org="acme", Tier 2 plugins should also be available.
	assert.Contains(t, langs, "python")
	assert.Contains(t, langs, "java")
	assert.Contains(t, langs, "typescript")
	assert.Contains(t, langs, "rust")
	assert.Contains(t, langs, "cpp")
}

func TestBuildSiteData_NoOrg_OnlyGoCoords(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "",
		Repo:    "apis",
		Modules: []catalog.Module{
			{
				ID:      "proto/payments/ledger/v1",
				Format:  "proto",
				Domain:  "payments",
				APILine: "v1",
			},
		},
	}

	// Without org, Tier 2 plugins (Python, Java, etc.) are not available.
	data := BuildSiteData(cat, "github.com/unknown/apis", "", "")

	require.Len(t, data.APIs, 1)
	langs := data.APIs[0].Languages

	assert.Contains(t, langs, "go")
	assert.NotContains(t, langs, "python")
	assert.NotContains(t, langs, "java")
}

func TestBuildSiteData_UnparseableID_Skipped(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []catalog.Module{
			{
				ID:     "bad-id",
				Format: "proto",
			},
			{
				ID:      "proto/payments/ledger/v1",
				Format:  "proto",
				Domain:  "payments",
				APILine: "v1",
			},
		},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "", "acme")

	// The bad-id module should be skipped; only the valid one should appear.
	assert.Len(t, data.APIs, 1)
	assert.Equal(t, "proto/payments/ledger/v1", data.APIs[0].ID)
}

func TestBuildSiteData_ExternalAPI(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []catalog.Module{
			{
				ID:        "proto/google/pubsub/v1",
				Format:    "proto",
				Domain:    "google",
				APILine:   "v1",
				Lifecycle: "stable",
				Origin:    "external",
			},
		},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "", "acme")

	require.Len(t, data.APIs, 1)
	assert.Equal(t, "external", data.APIs[0].Origin)
}

func TestBuildSiteData_MultipleFormats(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []catalog.Module{
			{
				ID:      "proto/payments/ledger/v1",
				Format:  "proto",
				Domain:  "payments",
				APILine: "v1",
			},
			{
				ID:      "openapi/billing/invoices/v2",
				Format:  "openapi",
				Domain:  "billing",
				APILine: "v2",
			},
			{
				ID:      "avro/events/clicks/v1",
				Format:  "avro",
				Domain:  "events",
				APILine: "v1",
			},
		},
	}

	data := BuildSiteData(cat, "github.com/acme/apis", "", "acme")

	assert.Len(t, data.APIs, 3)
	formats := make(map[string]bool)
	for _, api := range data.APIs {
		formats[api.Format] = true
	}
	assert.True(t, formats["proto"])
	assert.True(t, formats["openapi"])
	assert.True(t, formats["avro"])
}
