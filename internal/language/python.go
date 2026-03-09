package language

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/infobloxopen/apx/internal/ui"
)

func init() {
	Register(&pythonPlugin{})
}

type pythonPlugin struct{}

func (p *pythonPlugin) Name() string { return "python" }
func (p *pythonPlugin) Tier() int    { return 2 }

// Available returns true only when an org is configured.
func (p *pythonPlugin) Available(ctx DerivationContext) bool { return ctx.Org != "" }

func (p *pythonPlugin) DeriveCoords(ctx DerivationContext) (config.LanguageCoords, error) {
	return config.LanguageCoords{
		Module: config.DerivePythonDistName(ctx.Org, ctx.API),
		Import: config.DerivePythonImport(ctx.Org, ctx.API),
	}, nil
}

func (p *pythonPlugin) ReportLines(coords config.LanguageCoords) []ReportLine {
	return []ReportLine{
		{Label: "Py dist", Value: coords.Module},
		{Label: "Py import", Value: coords.Import},
	}
}

func (p *pythonPlugin) UnlinkHint(ctx DerivationContext) *UnlinkHint {
	return &UnlinkHint{
		Message: fmt.Sprintf("Python: Run 'pip install %s' to install the released package",
			config.DerivePythonDistName(ctx.Org, ctx.API)),
	}
}

// Scaffold implements Scaffolder — creates pyproject.toml and __init__.py hierarchy.
func (p *pythonPlugin) Scaffold(overlayPath string, ctx DerivationContext) error {
	distName := config.DerivePythonDistName(ctx.Org, ctx.API)
	importRoot := config.DerivePythonImport(ctx.Org, ctx.API)
	return overlay.ScaffoldPythonPackage(overlayPath, distName, importRoot)
}

// Link implements Linker — runs pip install -e for Python overlays.
func (p *pythonPlugin) Link(workDir, filterPath string) error {
	venv := os.Getenv("VIRTUAL_ENV")
	if venv == "" {
		return fmt.Errorf("no active virtualenv detected (VIRTUAL_ENV is not set)\nActivate a virtualenv first: source .venv/bin/activate")
	}

	pip := PipPath(venv)
	if _, err := os.Stat(pip); os.IsNotExist(err) {
		return fmt.Errorf("pip not found at %s — is the virtualenv valid?", pip)
	}

	mgr := overlay.NewManager(workDir)
	overlays, err := mgr.List()
	if err != nil {
		return fmt.Errorf("listing overlays: %w", err)
	}

	linked := 0
	for _, ov := range overlays {
		if ov.Language != "python" {
			continue
		}
		if filterPath != "" && ov.ModulePath != filterPath {
			continue
		}

		// Only link overlays that have a pyproject.toml (scaffolded).
		pyproject := filepath.Join(ov.Path, "pyproject.toml")
		if _, err := os.Stat(pyproject); os.IsNotExist(err) {
			ui.Warning("Skipping %s — no pyproject.toml (run 'apx gen python' first)", ov.ModulePath)
			continue
		}

		ui.Info("Linking %s ...", ov.ModulePath)
		installCmd := exec.Command(pip, "install", "-e", ov.Path)
		installCmd.Env = os.Environ()
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("pip install -e failed for %s: %w", ov.ModulePath, err)
		}
		linked++
	}

	if filterPath != "" && linked == 0 {
		cfg, _ := config.Load("")
		if cfg != nil && cfg.Org != "" {
			return fmt.Errorf("no Python overlay found for %s — run 'apx gen python' first", filterPath)
		}
		return fmt.Errorf("no Python overlay found for %s — ensure org is configured in apx.yaml and run 'apx gen python'", filterPath)
	}

	if linked == 0 {
		ui.Info("No Python overlays to link. Run 'apx gen python' first.")
		return nil
	}

	ui.Success("Linked %d Python overlay(s) in editable mode", linked)
	return nil
}

// PipPath returns the platform-appropriate path to pip inside a virtualenv.
func PipPath(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "pip.exe")
	}
	return filepath.Join(venvDir, "bin", "pip")
}
