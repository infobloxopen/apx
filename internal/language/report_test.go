package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFormatIdentityReport_AllLanguages(t *testing.T) {
	api := &config.APIIdentity{
		ID: "proto/payments/ledger/v1", Format: "proto",
		Domain: "payments", Name: "ledger", Line: "v1",
	}
	source := &config.SourceIdentity{Repo: "github.com/acme/apis", Path: "proto/payments/ledger/v1"}
	release := &config.ReleaseInfo{Current: "v1.0.0"}
	langs := map[string]config.LanguageCoords{
		"go":         {Module: "github.com/acme/apis/proto/payments/ledger", Import: "github.com/acme/apis/proto/payments/ledger/v1"},
		"python":     {Module: "acme-payments-ledger-v1", Import: "acme_apis.payments.ledger.v1"},
		"java":       {Module: "com.acme.apis:payments-ledger-v1-proto", Import: "com.acme.apis.payments.ledger.v1"},
		"typescript": {Module: "@acme/payments-ledger-v1-proto", Import: "@acme/payments-ledger-v1-proto"},
	}

	report := FormatIdentityReport(api, source, release, langs)

	// API section
	assert.Contains(t, report, "API:        proto/payments/ledger/v1")
	assert.Contains(t, report, "Format:     proto")
	assert.Contains(t, report, "Source:     github.com/acme/apis/proto/payments/ledger/v1")
	assert.Contains(t, report, "Release:    v1.0.0")
	assert.Contains(t, report, "Tag:        proto/payments/ledger/v1/v1.0.0")

	// Language section — all 4 languages
	assert.Contains(t, report, "Go module:  github.com/acme/apis/proto/payments/ledger")
	assert.Contains(t, report, "Go import:  github.com/acme/apis/proto/payments/ledger/v1")
	assert.Contains(t, report, "Py dist:    acme-payments-ledger-v1")
	assert.Contains(t, report, "Py import:  acme_apis.payments.ledger.v1")
	assert.Contains(t, report, "Maven:      com.acme.apis:payments-ledger-v1-proto")
	assert.Contains(t, report, "Java pkg:   com.acme.apis.payments.ledger.v1")
	assert.Contains(t, report, "npm:        @acme/payments-ledger-v1-proto")
}

func TestFormatIdentityReport_GoOnly(t *testing.T) {
	api := &config.APIIdentity{
		ID: "proto/orders/v1", Format: "proto", Name: "orders", Line: "v1",
	}
	langs := map[string]config.LanguageCoords{
		"go": {Module: "github.com/acme/apis/proto/orders", Import: "github.com/acme/apis/proto/orders/v1"},
	}

	report := FormatIdentityReport(api, nil, nil, langs)

	assert.Contains(t, report, "Go module:")
	assert.NotContains(t, report, "Py dist:")
	assert.NotContains(t, report, "Maven:")
	assert.NotContains(t, report, "npm:")
}

func TestFormatIdentityReport_WithLifecycle(t *testing.T) {
	api := &config.APIIdentity{
		ID: "proto/orders/v1", Format: "proto", Name: "orders", Line: "v1",
		Lifecycle: "beta",
	}
	langs := map[string]config.LanguageCoords{
		"go": {Module: "mod", Import: "imp"},
	}

	report := FormatIdentityReport(api, nil, nil, langs)
	assert.Contains(t, report, "Lifecycle:  beta")
}

func TestFormatLanguageLines(t *testing.T) {
	langs := map[string]config.LanguageCoords{
		"go":     {Module: "go-mod", Import: "go-imp"},
		"python": {Module: "py-dist", Import: "py-imp"},
	}

	lines := FormatLanguageLines(langs)
	assert.Contains(t, lines, "Go module:  go-mod")
	assert.Contains(t, lines, "Py dist:    py-dist")
}
