// Command docgen assembles documentation fragments from language plugins
// into generated include files for the Sphinx documentation.
//
// Usage:
//
//	go run ./cmd/docgen
//	go generate ./internal/language/...
//
// Generated files are written to docs/_generated/ and should NOT be
// committed to version control. They are built as part of the docs
// pipeline and during `go generate`.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// Import language package to trigger all init() registrations.
	"github.com/infobloxopen/apx/internal/language"
)

func main() {
	// Determine module root so docgen works whether run from repo root
	// or via go generate from any package directory.
	root := findModuleRoot()
	outDir := filepath.Join(root, "docs", "_generated")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "docgen: creating output dir: %v\n", err)
		os.Exit(1)
	}

	// Collect DocMeta from all plugins that implement DocContributor.
	var docs []pluginDoc
	for _, p := range language.All() {
		if dc, ok := p.(language.DocContributor); ok {
			docs = append(docs, pluginDoc{name: p.Name(), meta: dc.DocMeta()})
		}
	}

	if len(docs) == 0 {
		fmt.Fprintln(os.Stderr, "docgen: no plugins implement DocContributor")
		os.Exit(1)
	}

	// 1. Language Support Matrix
	if err := writeSupportMatrix(outDir, docs); err != nil {
		fmt.Fprintf(os.Stderr, "docgen: %v\n", err)
		os.Exit(1)
	}

	// 2. Identity Derivation Table
	if err := writeIdentityTable(outDir, docs); err != nil {
		fmt.Fprintf(os.Stderr, "docgen: %v\n", err)
		os.Exit(1)
	}

	// 3. Per-language code generation and dev workflow sections
	for _, d := range docs {
		if content, ok := d.meta.Sections["code_generation"]; ok {
			path := filepath.Join(outDir, fmt.Sprintf("code-gen-%s.md", d.name))
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "docgen: writing %s: %v\n", path, err)
				os.Exit(1)
			}
		}
		if content, ok := d.meta.Sections["dev_workflow"]; ok {
			path := filepath.Join(outDir, fmt.Sprintf("dev-workflow-%s.md", d.name))
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "docgen: writing %s: %v\n", path, err)
				os.Exit(1)
			}
		}
	}

	// 4. Per-language path mapping tables
	for _, d := range docs {
		if len(d.meta.PathMappings) == 0 {
			continue
		}
		path := filepath.Join(outDir, fmt.Sprintf("path-mapping-%s.md", d.name))
		if err := writePathMappings(path, d.name, d.meta.PathMappings); err != nil {
			fmt.Fprintf(os.Stderr, "docgen: writing %s: %v\n", path, err)
			os.Exit(1)
		}
	}

	fmt.Printf("docgen: wrote %d generated doc includes to %s/\n", countFiles(outDir), outDir)
}

func writeSupportMatrix(outDir string, docs []pluginDoc) error {
	var sb strings.Builder
	sb.WriteString("| Language | Artifact | Local Overlay | Resolution | Codegen | Dev Command | Unlink Hint | Tier |\n")
	sb.WriteString("|----------|----------|---------------|------------|---------|-------------|-------------|------|\n")

	for _, d := range docs {
		m := d.meta.SupportMatrix
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			strings.Title(d.name),
			m["published_artifact"],
			m["local_overlay"],
			m["resolution"],
			m["codegen"],
			m["dev_command"],
			m["unlink_hint"],
			m["tier"],
		))
	}

	return os.WriteFile(filepath.Join(outDir, "language-support-matrix.md"), []byte(sb.String()), 0o644)
}

func writeIdentityTable(outDir string, docs []pluginDoc) error {
	var sb strings.Builder
	sb.WriteString("| Coordinate | Derived Value |\n")
	sb.WriteString("|------------|---------------|\n")

	for _, d := range docs {
		for _, row := range d.meta.IdentityRows {
			sb.WriteString(fmt.Sprintf("| %s | `%s` |\n", row.CoordType, row.DerivedValue))
		}
	}

	return os.WriteFile(filepath.Join(outDir, "identity-derivation-table.md"), []byte(sb.String()), 0o644)
}

func writePathMappings(path, langName string, mappings []language.PathMapping) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("#### %s Path Mappings\n\n", strings.Title(langName)))
	sb.WriteString("| APX Path | Target Coordinate | Description |\n")
	sb.WriteString("|----------|-------------------|-------------|\n")

	for _, m := range mappings {
		sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", m.APXPath, m.TargetCoord, m.Description))
	}

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

type pluginDoc struct {
	name string
	meta language.DocMeta
}

// findModuleRoot walks up from cwd looking for go.mod.
func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "." // fallback to cwd
		}
		dir = parent
	}
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}
