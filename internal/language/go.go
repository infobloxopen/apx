package language

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/overlay"
)

func init() {
	Register(&goPlugin{})
}

type goPlugin struct{}

func (g *goPlugin) Name() string { return "go" }
func (g *goPlugin) Tier() int    { return 1 }

// Available always returns true — Go is always available regardless of org.
func (g *goPlugin) Available(ctx DerivationContext) bool { return true }

func (g *goPlugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error) {
	goRoot := config.EffectiveGoRoot(ctx.SourceRepo, ctx.ImportRoot)
	goMod, err := config.DeriveGoModule(goRoot, ctx.API)
	if err != nil {
		return config.LanguageCoords{}, fmt.Errorf("deriving Go module: %w", err)
	}
	goImport, err := config.DeriveGoImport(goRoot, ctx.API)
	if err != nil {
		return config.LanguageCoords{}, fmt.Errorf("deriving Go import: %w", err)
	}
	return config.LanguageCoords{Module: goMod, Import: goImport}, nil
}

func (g *goPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	return []ReportLine{
		{Label: "Go module", Value: coords.Module},
		{Label: "Go import", Value: coords.Import},
	}
}

func (g *goPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	goRoot := config.EffectiveGoRoot(ctx.SourceRepo, ctx.ImportRoot)
	return &UnlinkHint{
		Message: fmt.Sprintf("Go: Run 'go get %s/%s' to add released module", goRoot, ctx.API.ID),
	}
}

// PostGen implements PostGenHook — runs go.work sync after Go code generation.
func (g *goPlugin) PostGen(workDir string) error {
	mgr := overlay.NewManager(workDir)
	return mgr.Sync()
}
