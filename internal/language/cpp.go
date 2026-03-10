package language

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
)

func init() {
	Register(&cppPlugin{})
}

type cppPlugin struct{}

func (c *cppPlugin) Name() string { return "cpp" }
func (c *cppPlugin) Tier() int    { return 2 }

// Available returns true only when an org is configured.
func (c *cppPlugin) Available(ctx DerivationContext) bool { return ctx.Org != "" }

func (c *cppPlugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error) {
	return config.LanguageCoords{
		Module: deriveCppConanRef(ctx.Org, ctx.API),
		Import: deriveCppNamespace(ctx.Org, ctx.API),
	}, nil
}

func (c *cppPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	return []ReportLine{
		{Label: "Conan", Value: coords.Module},
		{Label: "C++ ns", Value: coords.Import},
	}
}

func (c *cppPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	ref := deriveCppConanRef(ctx.Org, ctx.API)
	return &UnlinkHint{
		Message: fmt.Sprintf("C++: Add %s/<version> to your conanfile", ref),
	}
}

// ---------------------------------------------------------------------------
// C++ / Conan identity derivation (private to this plugin)
// ---------------------------------------------------------------------------

// deriveCppConanRef computes the Conan package reference for a C++ overlay.
//
// Rules:
//   - Pattern: <org>-<domain>-<name>-<line>-proto (4-part) or <org>-<name>-<line>-proto (3-part)
//   - All lowercase, joined with hyphens, -proto suffix
//   - Example: org="acme", proto/payments/ledger/v1 → "acme-payments-ledger-v1-proto"
//   - Example: org="acme", proto/orders/v1 (3-part) → "acme-orders-v1-proto"
func deriveCppConanRef(org string, api *config.APIIdentity) string {
	parts := []string{strings.ToLower(org)}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	parts = append(parts, "proto")
	return strings.Join(parts, "-")
}

// deriveCppNamespace computes the C++ namespace for generated code.
//
// Rules:
//   - Pattern: <org>::<domain>::<name>::<line> (4-part) or <org>::<name>::<line> (3-part)
//   - All lowercase, joined with ::
//   - Example: org="acme", proto/payments/ledger/v1 → "acme::payments::ledger::v1"
//   - Example: org="Acme-Corp", proto/payments/ledger/v1 → "acme_corp::payments::ledger::v1"
//   - Example: org="acme", proto/orders/v1 (3-part) → "acme::orders::v1"
func deriveCppNamespace(org string, api *config.APIIdentity) string {
	// C++ identifiers cannot contain hyphens; replace with underscores.
	parts := []string{strings.ReplaceAll(strings.ToLower(org), "-", "_")}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	return strings.Join(parts, "::")
}
