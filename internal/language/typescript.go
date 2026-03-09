package language

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
)

func init() {
	Register(&typescriptPlugin{})
}

type typescriptPlugin struct{}

func (ts *typescriptPlugin) Name() string { return "typescript" }
func (ts *typescriptPlugin) Tier() int    { return 2 }

// Available returns true only when an org is configured.
func (ts *typescriptPlugin) Available(ctx DerivationContext) bool { return ctx.Org != "" }

func (ts *typescriptPlugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error) {
	npmPkg := deriveNpmPackage(ctx.Org, ctx.API)
	return config.LanguageCoords{
		Module: npmPkg,
		Import: npmPkg, // TypeScript import path == npm package name
	}, nil
}

func (ts *typescriptPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	// TypeScript module == import, so we only show one line.
	return []ReportLine{
		{Label: "npm", Value: coords.Module},
	}
}

func (ts *typescriptPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	npmPkg := deriveNpmPackage(ctx.Org, ctx.API)
	return &UnlinkHint{
		Message: fmt.Sprintf("TypeScript: Run 'npm install %s' to install the released package", npmPkg),
	}
}

// ---------------------------------------------------------------------------
// TypeScript / npm identity derivation (private to this plugin)
// ---------------------------------------------------------------------------

// deriveNpmPackage computes the scoped npm package name for an API.
//
// Rules:
//   - Pattern: @<org>/<domain>-<name>-<line>-proto (4-part) or @<org>/<name>-<line>-proto (3-part)
//   - Lowercased, hyphens join path segments, -proto suffix
//   - Example: org="acme", proto/payments/ledger/v1 → "@acme/payments-ledger-v1-proto"
//   - Example: org="acme", proto/orders/v1 (3-part) → "@acme/orders-v1-proto"
func deriveNpmPackage(org string, api *config.APIIdentity) string {
	scope := strings.ToLower(org)
	var parts []string
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	parts = append(parts, "proto")
	return "@" + scope + "/" + strings.Join(parts, "-")
}
