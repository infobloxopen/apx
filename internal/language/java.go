package language

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
)

func init() {
	Register(&javaPlugin{})
}

type javaPlugin struct{}

func (j *javaPlugin) Name() string { return "java" }
func (j *javaPlugin) Tier() int    { return 2 }

// Available returns true only when an org is configured.
func (j *javaPlugin) Available(ctx DerivationContext) bool { return ctx.Org != "" }

func (j *javaPlugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error) {
	return config.LanguageCoords{
		Module: deriveMavenCoords(ctx.Org, ctx.API),
		Import: deriveJavaPackage(ctx.Org, ctx.API),
	}, nil
}

func (j *javaPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	return []ReportLine{
		{Label: "Maven", Value: coords.Module},
		{Label: "Java pkg", Value: coords.Import},
	}
}

func (j *javaPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	coords := deriveMavenCoords(ctx.Org, ctx.API)
	return &UnlinkHint{
		Message: fmt.Sprintf("Java: Add %s:<version> to your pom.xml", coords),
	}
}

// ---------------------------------------------------------------------------
// Java / Maven identity derivation (private to this plugin)
// ---------------------------------------------------------------------------

// deriveMavenGroupId computes the Maven groupId for an organization.
//
// Rules:
//   - Pattern: com.<org>.apis
//   - Lowercased, hyphens replaced with dots
//   - Example: org="acme" → "com.acme.apis"
//   - Example: org="Acme-Corp" → "com.acme.corp.apis"
func deriveMavenGroupId(org string) string {
	normalized := strings.ToLower(org)
	normalized = strings.ReplaceAll(normalized, "-", ".")
	return "com." + normalized + ".apis"
}

// deriveMavenArtifactId computes the Maven artifactId for an API.
//
// Rules:
//   - Combines domain (if present), name, and line with hyphens
//   - Appends "-proto" suffix to distinguish schema artifacts
//   - Example: proto/payments/ledger/v1 → "payments-ledger-v1-proto"
//   - Example: proto/orders/v1 (3-part) → "orders-v1-proto"
func deriveMavenArtifactId(api *config.APIIdentity) string {
	var parts []string
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	parts = append(parts, "proto")
	return strings.Join(parts, "-")
}

// deriveMavenCoords returns the full Maven coordinate string (groupId:artifactId).
//
// Example: org="acme", proto/payments/ledger/v1 → "com.acme.apis:payments-ledger-v1-proto"
func deriveMavenCoords(org string, api *config.APIIdentity) string {
	return deriveMavenGroupId(org) + ":" + deriveMavenArtifactId(api)
}

// deriveJavaPackage computes the Java package name for an API.
//
// Rules:
//   - Pattern: com.<org>.apis.<domain>.<name>.<line>
//   - Lowercased, hyphens replaced with dots
//   - Example: org="acme", proto/payments/ledger/v1 → "com.acme.apis.payments.ledger.v1"
//   - Example: org="acme", proto/orders/v1 → "com.acme.apis.orders.v1"
func deriveJavaPackage(org string, api *config.APIIdentity) string {
	normalized := strings.ToLower(org)
	normalized = strings.ReplaceAll(normalized, "-", ".")
	parts := []string{"com", normalized, "apis"}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	return strings.Join(parts, ".")
}
