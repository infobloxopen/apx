package language

import (
	"fmt"

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
	npmPkg := config.DeriveNpmPackage(ctx.Org, ctx.API)
	return config.LanguageCoords{
		Module: npmPkg,
		Import: config.DeriveTsImport(ctx.Org, ctx.API),
	}, nil
}

func (ts *typescriptPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	// TypeScript module == import, so we only show one line.
	return []ReportLine{
		{Label: "npm", Value: coords.Module},
	}
}

func (ts *typescriptPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	npmPkg := config.DeriveNpmPackage(ctx.Org, ctx.API)
	return &UnlinkHint{
		Message: fmt.Sprintf("TypeScript: Run 'npm install %s' to install the released package", npmPkg),
	}
}
