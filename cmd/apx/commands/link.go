package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/overlay"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <language> [module-path]",
		Short: "Link overlays for local development",
		Long: `Link generated overlays for local development.

For Python: runs 'pip install -e' for each overlay in the active virtualenv.
For Go: use 'apx sync' instead (Go uses go.work overlays).

Examples:
  apx link python                              # link all Python overlays
  apx link python proto/payments/ledger/v1     # link a specific overlay`,
		Args: cobra.RangeArgs(1, 2),
		RunE: linkAction,
	}
	return cmd
}

func linkAction(cmd *cobra.Command, args []string) error {
	lang := args[0]
	var filterPath string
	if len(args) > 1 {
		filterPath = args[1]
	}

	switch lang {
	case "go":
		ui.Info("Go uses go.work overlays — run 'apx sync' instead.")
		return nil
	case "python":
		return linkPython(filterPath)
	default:
		return fmt.Errorf("unsupported language for link: %s (supported: python)", lang)
	}
}

func linkPython(filterPath string) error {
	venv := os.Getenv("VIRTUAL_ENV")
	if venv == "" {
		return fmt.Errorf("no active virtualenv detected (VIRTUAL_ENV is not set)\nActivate a virtualenv first: source .venv/bin/activate")
	}

	pip := pipPath(venv)
	if _, err := os.Stat(pip); os.IsNotExist(err) {
		return fmt.Errorf("pip not found at %s — is the virtualenv valid?", pip)
	}

	mgr := overlay.NewManager(".")
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
		// Check if the user needs to run gen first.
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

// pipPath returns the platform-appropriate path to pip inside a virtualenv.
func pipPath(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "pip.exe")
	}
	return filepath.Join(venvDir, "bin", "pip")
}
