package commands

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/language"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	// Build supported languages dynamically from plugins that implement Linker.
	var linkable []string
	for _, p := range language.All() {
		if _, ok := p.(language.Linker); ok {
			linkable = append(linkable, p.Name())
		}
	}
	supported := strings.Join(linkable, ", ")

	cmd := &cobra.Command{
		Use:   "link <language> [module-path]",
		Short: "Link overlays for local development",
		Long: fmt.Sprintf(`Link generated overlays for local development.

For Python: runs 'pip install -e' for each overlay in the active virtualenv.
For Go: use 'apx sync' instead (Go uses go.work overlays).

Supported languages: %s

Examples:
  apx link python                              # link all Python overlays
  apx link python proto/payments/ledger/v1     # link a specific overlay`, supported),
		Args: cobra.RangeArgs(1, 2),
		RunE: linkAction,
	}
	return cmd
}

func linkableNames() []string {
	var names []string
	for _, p := range language.All() {
		if _, ok := p.(language.Linker); ok {
			names = append(names, p.Name())
		}
	}
	return names
}

func linkAction(cmd *cobra.Command, args []string) error {
	lang := args[0]
	var filterPath string
	if len(args) > 1 {
		filterPath = args[1]
	}

	plugin := language.Get(lang)
	if plugin == nil {
		return fmt.Errorf("unknown language %q (supported: %s)", lang, strings.Join(language.Names(), ", "))
	}

	linker, ok := plugin.(language.Linker)
	if !ok {
		if lang == "go" {
			ui.Info("%s does not support 'link'. Use 'apx sync' for Go overlays.", lang)
			return nil
		}
		return fmt.Errorf("unsupported language %q for link (supported: %s)", lang, strings.Join(linkableNames(), ", "))
	}

	return linker.Link(".", filterPath)
}
