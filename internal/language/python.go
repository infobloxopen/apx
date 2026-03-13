package language

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

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
		Module: derivePythonDistName(ctx.Org, ctx.API),
		Import: derivePythonImport(ctx.Org, ctx.API),
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
			derivePythonDistName(ctx.Org, ctx.API)),
	}
}

// Scaffold implements Scaffolder — creates pyproject.toml and __init__.py hierarchy.
func (p *pythonPlugin) Scaffold(overlayPath string, ctx DerivationContext) error {
	distName := derivePythonDistName(ctx.Org, ctx.API)
	importRoot := derivePythonImport(ctx.Org, ctx.API)
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

// Unlink implements Unlinker — runs pip uninstall for Python overlays.
func (p *pythonPlugin) Unlink(workDir, filterPath string) error {
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

	unlinked := 0
	for _, ov := range overlays {
		if ov.Language != "python" {
			continue
		}
		if filterPath != "" && ov.ModulePath != filterPath {
			continue
		}

		pyproject := filepath.Join(ov.Path, "pyproject.toml")
		distName, err := readPyprojectName(pyproject)
		if err != nil {
			ui.Warning("Skipping %s — could not read pyproject.toml: %v", ov.ModulePath, err)
			continue
		}

		ui.Info("Unlinking %s ...", ov.ModulePath)
		uninstallCmd := exec.Command(pip, "uninstall", "-y", distName)
		uninstallCmd.Env = os.Environ()
		uninstallCmd.Stdout = os.Stdout
		uninstallCmd.Stderr = os.Stderr
		if err := uninstallCmd.Run(); err != nil {
			return fmt.Errorf("pip uninstall failed for %s: %w", ov.ModulePath, err)
		}
		unlinked++
	}

	if unlinked == 0 {
		ui.Info("No linked Python overlays found.")
		return nil
	}

	ui.Success("Unlinked %d Python overlay(s)", unlinked)
	return nil
}

// readPyprojectName extracts the package name from a pyproject.toml file.
func readPyprojectName(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), `"`), nil
			}
		}
	}
	return "", fmt.Errorf("name field not found in %s", path)
}

// PipPath returns the platform-appropriate path to pip inside a virtualenv.
func PipPath(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "pip.exe")
	}
	return filepath.Join(venvDir, "bin", "pip")
}

// ---------------------------------------------------------------------------
// Python identity derivation (private to this plugin)
// ---------------------------------------------------------------------------

// derivePythonDistName computes a PEP 625 distribution name for a Python package.
//
// Rules:
//   - Combines org, domain (if present), API name, and line
//   - All lowercase, joined with hyphens
//   - Example: org="acme", proto/payments/ledger/v1 → "acme-payments-ledger-v1"
//   - Example: org="acme", proto/orders/v1 (3-part, no domain) → "acme-orders-v1"
func derivePythonDistName(org string, api *config.APIIdentity) string {
	parts := []string{strings.ToLower(org)}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	return strings.Join(parts, "-")
}

// derivePythonImport computes a dotted Python import path for an API.
//
// Rules:
//   - Top-level namespace: {org}_apis (underscore-joined, Python identifier safe)
//   - Sub-packages: domain (if present), name, line
//   - Example: org="acme", proto/payments/ledger/v1 → "acme_apis.payments.ledger.v1"
//   - Example: org="Acme-Corp", proto/payments/ledger/v1 → "acme_corp_apis.payments.ledger.v1"
//   - Example: org="acme", proto/orders/v1 → "acme_apis.orders.v1"
func derivePythonImport(org string, api *config.APIIdentity) string {
	// Python identifiers cannot contain hyphens; replace with underscores.
	namespace := strings.ReplaceAll(strings.ToLower(org), "-", "_") + "_apis"
	parts := []string{namespace}
	if api.Domain != "" {
		parts = append(parts, strings.ToLower(api.Domain))
	}
	parts = append(parts, strings.ToLower(api.Name))
	parts = append(parts, strings.ToLower(api.Line))
	return strings.Join(parts, ".")
}

// pep440PreRe matches SemVer pre-release tags: alpha, beta, rc with optional dot-separator.
var pep440PreRe = regexp.MustCompile(`-(alpha|beta|rc)\.?(\d+)`)

// NormalizePEP440Version converts a SemVer version string to PEP 440 format.
//
// Rules:
//   - Strips leading "v" prefix
//   - Converts -alpha.N → aN
//   - Converts -beta.N → bN
//   - Converts -rc.N → rcN
//   - Example: "v1.2.3" → "1.2.3"
//   - Example: "v1.0.0-beta.1" → "1.0.0b1"
func NormalizePEP440Version(semver string) string {
	v := strings.TrimPrefix(semver, "v")

	v = pep440PreRe.ReplaceAllStringFunc(v, func(match string) string {
		sub := pep440PreRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		tag, num := sub[1], sub[2]
		switch tag {
		case "alpha":
			return "a" + num
		case "beta":
			return "b" + num
		case "rc":
			return "rc" + num
		}
		return match
	})

	return v
}
