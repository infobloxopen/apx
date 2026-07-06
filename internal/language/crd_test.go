package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
)

// A CRD carries no language bindings: apx lifecycles it, controller-gen /
// kubebuilder generate it. No language coordinate should be derived.
func TestCRDHasNoLanguageCoords(t *testing.T) {
	ctx := DerivationContext{
		SourceRepo: "github.com/acme/apis",
		Org:        "acme",
		API: &config.APIIdentity{
			ID:     "crd/appkit.infoblox.dev/appcontract/v1alpha1",
			Format: "crd",
			Domain: "appkit.infoblox.dev",
			Name:   "appcontract",
			Line:   "v1alpha1",
		},
	}
	if plugins := Available(ctx); len(plugins) != 0 {
		names := make([]string, len(plugins))
		for i, p := range plugins {
			names[i] = p.Name()
		}
		t.Errorf("expected no language plugins for crd, got %v", names)
	}
	coords, err := DeriveAllCoords(ctx)
	if err != nil {
		t.Fatalf("DeriveAllCoords: %v", err)
	}
	if len(coords) != 0 {
		t.Errorf("expected no coords for crd, got %v", coords)
	}
}

// A non-crd format still derives Go coordinates (guards against over-gating).
func TestProtoStillHasGoCoords(t *testing.T) {
	ctx := DerivationContext{
		SourceRepo: "github.com/acme/apis",
		Org:        "acme",
		API: &config.APIIdentity{
			ID:     "proto/payments/ledger/v1",
			Format: "proto",
			Domain: "payments",
			Name:   "ledger",
			Line:   "v1",
		},
	}
	coords, err := DeriveAllCoords(ctx)
	if err != nil {
		t.Fatalf("DeriveAllCoords: %v", err)
	}
	if _, ok := coords["go"]; !ok {
		t.Errorf("expected go coords for proto, got %v", coords)
	}
}
