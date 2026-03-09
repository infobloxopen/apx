package language

import (
	"fmt"
	"path"

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
	goMod, err := deriveGoModule(goRoot, ctx.API)
	if err != nil {
		return config.LanguageCoords{}, fmt.Errorf("deriving Go module: %w", err)
	}
	goImport, err := deriveGoImport(goRoot, ctx.API)
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

// ---------------------------------------------------------------------------
// Go identity derivation (private to this plugin)
// ---------------------------------------------------------------------------

// deriveGoModule computes the Go module path for the given API line.
//
// Rules (per Go module versioning):
//   - For v0: <sourceRepo>/<format>/<domain>/<name>       (no version suffix)
//   - For v1: <sourceRepo>/<format>/<domain>/<name>       (no version suffix)
//   - For v2+: <sourceRepo>/<format>/<domain>/<name>/v<N>  (major version suffix)
func deriveGoModule(sourceRepo string, api *config.APIIdentity) (string, error) {
	major, err := config.LineMajor(api.Line)
	if err != nil {
		return "", err
	}

	base := path.Join(sourceRepo, api.Format, api.Domain, api.Name)
	if major <= 1 {
		return base, nil
	}
	return fmt.Sprintf("%s/v%d", base, major), nil
}

// deriveGoImport computes the Go import path for the given API line.
//
// Rules:
//   - For v0: <sourceRepo>/<format>/<domain>/<name>/v0      (v0 in import path)
//   - For v1: <sourceRepo>/<format>/<domain>/<name>/v1      (v1 in import path)
//   - For v2+: <sourceRepo>/<format>/<domain>/<name>/v<N>    (same as module path)
func deriveGoImport(sourceRepo string, api *config.APIIdentity) (string, error) {
	major, err := config.LineMajor(api.Line)
	if err != nil {
		return "", err
	}

	base := path.Join(sourceRepo, api.Format, api.Domain, api.Name)
	return fmt.Sprintf("%s/v%d", base, major), nil
}
