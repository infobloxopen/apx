package language

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
)

func init() {
	Register(&rustPlugin{})
}

type rustPlugin struct{}

func (r *rustPlugin) Name() string { return "rust" }
func (r *rustPlugin) Tier() int    { return 2 }

// Available returns true only when an org is configured.
func (r *rustPlugin) Available(ctx DerivationContext) bool { return ctx.Org != "" }

func (r *rustPlugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error) {
	return config.LanguageCoords{
		Module: deriveRustCrate(ctx.Org, ctx.API),
		Import: deriveRustModule(ctx.Org, ctx.API),
	}, nil
}

func (r *rustPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	return []ReportLine{
		{Label: "Crate", Value: coords.Module},
		{Label: "Rust mod", Value: coords.Import},
	}
}

func (r *rustPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	crate := deriveRustCrate(ctx.Org, ctx.API)
	return &UnlinkHint{
		Message: fmt.Sprintf("Rust: Add %s = \"<version>\" to your Cargo.toml", crate),
	}
}

// ---------------------------------------------------------------------------
// Rust / Cargo identity derivation (private to this plugin)
// ---------------------------------------------------------------------------

// deriveRustCrate computes the Cargo crate name for a Rust overlay.
//
// Rules:
//   - Pattern: <org>-<domain>-<name>-<line>-proto (4-part) or <org>-<name>-<line>-proto (3-part)
//   - All lowercase, joined with hyphens, -proto suffix
//   - Example: org="acme", proto/payments/ledger/v1 → "acme-payments-ledger-v1-proto"
//   - Example: org="acme", proto/orders/v1 (3-part) → "acme-orders-v1-proto"
func deriveRustCrate(org string, api *config.APIIdentity) string {
	parts := []string{strings.ToLower(org)}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	parts = append(parts, "proto")
	return strings.Join(parts, "-")
}

// deriveRustModule computes the Rust module path for generated code.
//
// Rules:
//   - Root module: <org>_<domain> (4-part) or <org>_<name> (3-part, no domain)
//   - Sub-modules: <name>::<line> (4-part) or <line> (3-part)
//   - All lowercase, underscore for crate root, :: for module path
//   - Example: org="acme", proto/payments/ledger/v1 → "acme_payments::ledger::v1"
//   - Example: org="Acme-Corp", proto/payments/ledger/v1 → "acme_corp_payments::ledger::v1"
//   - Example: org="acme", proto/orders/v1 (3-part) → "acme_orders::v1"
func deriveRustModule(org string, api *config.APIIdentity) string {
	// Rust identifiers cannot contain hyphens; replace with underscores.
	orgLower := strings.ReplaceAll(strings.ToLower(org), "-", "_")
	if api.Domain != "" {
		// 4-part: acme_payments::ledger::v1
		root := orgLower + "_" + strings.ToLower(api.Domain)
		return root + "::" + strings.ToLower(api.Name) + "::" + strings.ToLower(api.Line)
	}
	// 3-part: acme_orders::v1
	root := orgLower + "_" + strings.ToLower(api.Name)
	return root + "::" + strings.ToLower(api.Line)
}
