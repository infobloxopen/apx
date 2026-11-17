package commands

import (
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/publisher"
)

func TestReleaseTagMessage_Audience(t *testing.T) {
	m := &publisher.ReleaseManifest{
		APIID: "openapi/payments/ledger/v1", RequestedVersion: "v1.2.0", Lifecycle: "stable",
		SourceRepo: "github.com/acme/ledger", SourcePath: "api",
	}
	if msg := releaseTagMessage(m); strings.Contains(msg, "Audience") || strings.Contains(msg, "Tags") {
		t.Errorf("a release without tags and audience must not record them:\n%s", msg)
	}

	m.Audience = "internal"
	m.Tags = []string{"product:ddi"}
	msg := releaseTagMessage(m)
	for _, want := range []string{"Release openapi/payments/ledger/v1 v1.2.0", "\nLifecycle: stable", "\nTags: product:ddi", "\nAudience: internal"} {
		if !strings.Contains(msg, want) {
			t.Errorf("tag message does not contain %q:\n%s", want, msg)
		}
	}
}

func TestUpsertCatalogModule_Audience(t *testing.T) {
	manifest := func(version, audience string) *publisher.ReleaseManifest {
		return &publisher.ReleaseManifest{
			APIID: "openapi/payments/ledger/v1", Format: "openapi", Domain: "payments", Line: "v1",
			RequestedVersion: version, Lifecycle: "stable", CanonicalPath: "openapi/payments/ledger/v1",
			Audience: audience,
		}
	}
	cat := &catalog.Catalog{Version: 1}

	// A new entry without an audience has none.
	upsertCatalogModule(cat, manifest("v1.0.0", ""))
	if len(cat.Modules) != 1 || cat.Modules[0].Audience != "" {
		t.Fatalf("new entry: %+v", cat.Modules)
	}

	// A release with an audience sets it on the existing entry.
	upsertCatalogModule(cat, manifest("v1.1.0", "internal"))
	if len(cat.Modules) != 1 || cat.Modules[0].Audience != "internal" || cat.Modules[0].Version != "v1.1.0" {
		t.Fatalf("after v1.1.0: %+v", cat.Modules)
	}

	// A later release without an audience keeps the audience of the entry.
	upsertCatalogModule(cat, manifest("v1.2.0", ""))
	if cat.Modules[0].Audience != "internal" || cat.Modules[0].LatestStable != "v1.2.0" {
		t.Fatalf("after v1.2.0: %+v", cat.Modules[0])
	}

	// A release with another audience replaces it.
	upsertCatalogModule(cat, manifest("v1.3.0", "public"))
	if cat.Modules[0].Audience != "public" {
		t.Fatalf("after v1.3.0: %+v", cat.Modules[0])
	}
}
