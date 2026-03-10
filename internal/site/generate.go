package site

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

// Generate writes a complete static site to outputDir.
//
// It writes:
//   - data/index.json  — the full catalog data for client-side search/display
//   - index.html       — the single-page app shell
//   - assets/app.js    — the frontend JavaScript
//   - assets/style.css — the CSS styles
func Generate(data *SiteData, outputDir, basePath string) error {
	// Normalize base path: ensure leading slash, no trailing slash.
	basePath = "/" + strings.Trim(basePath, "/")
	if basePath == "/" {
		basePath = ""
	}

	// 1. Write data/index.json
	dataDir := filepath.Join(outputDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling site data: %w", err)
	}

	indexJSON := filepath.Join(dataDir, "index.json")
	if err := os.WriteFile(indexJSON, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("writing index.json: %w", err)
	}

	// 2. Copy embedded static assets to output directory.
	err = fs.WalkDir(staticFS, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip the "static/" prefix to get the relative output path.
		relPath := strings.TrimPrefix(path, "static")
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			return nil // root "static" directory itself
		}

		outPath := filepath.Join(outputDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}

		content, err := staticFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}

		// Template the base path into HTML files.
		if strings.HasSuffix(path, ".html") {
			content = []byte(strings.ReplaceAll(string(content), "{{BASE_PATH}}", basePath))
		}

		return os.WriteFile(outPath, content, 0o644)
	})
	if err != nil {
		return fmt.Errorf("copying static assets: %w", err)
	}

	return nil
}
