package language

import (
	"fmt"

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
		Module: config.DeriveMavenCoords(ctx.Org, ctx.API),
		Import: config.DeriveJavaPackage(ctx.Org, ctx.API),
	}, nil
}

func (j *javaPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	return []ReportLine{
		{Label: "Maven", Value: coords.Module},
		{Label: "Java pkg", Value: coords.Import},
	}
}

func (j *javaPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	coords := config.DeriveMavenCoords(ctx.Org, ctx.API)
	return &UnlinkHint{
		Message: fmt.Sprintf("Java: Add %s:<version> to your pom.xml", coords),
	}
}
