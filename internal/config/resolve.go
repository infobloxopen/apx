package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveAPIPath resolves an API ID or filesystem path to a validated
// filesystem path. If the argument is already a valid path it is returned
// as-is. If it parses as an API ID (format/domain/name/line), the function
// searches module_roots from the config (if provided), then falls back to
// common directory patterns.
//
// The returned path is always absolute.
func ResolveAPIPath(arg string, cfg *Config) (string, error) {
	// 1. If it already exists on disk, return it directly.
	abs, err := filepath.Abs(arg)
	if err == nil {
		if _, statErr := os.Stat(abs); statErr == nil {
			return abs, nil
		}
	}

	// 2. Try to parse as a strict 4-part API ID (format/domain/name/line).
	// If it parses, use module_roots and well-known fallback patterns.
	// If it doesn't parse (e.g. a 3-part simplified id like "proto/tagtest/v1"),
	// still try the common fallbacks below before giving up.
	var relPath string
	api, parseErr := ParseAPIID(arg)
	if parseErr == nil {
		relPath = DeriveSourcePath(api.ID)

		// 3. Search module_roots from config.
		if cfg != nil {
			for _, root := range cfg.ModuleRoots {
				candidate, _ := filepath.Abs(filepath.Join(root, relPath))
				if candidate != "" {
					if _, statErr := os.Stat(candidate); statErr == nil {
						return candidate, nil
					}
				}
			}
		}
	} else {
		// Arg is not a valid 4-part API ID; use it verbatim as the relative path.
		relPath = arg
	}

	// 4. Fall back to common patterns relative to cwd.
	fallbacks := []string{
		relPath,
		filepath.Join("internal", "apis", relPath),
		filepath.Join("schemas", relPath),
		filepath.Join("api", relPath),
	}
	for _, fb := range fallbacks {
		candidate, _ := filepath.Abs(fb)
		if candidate != "" {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("path does not exist: %s", arg)
}

// ResolveAPIFormat extracts the schema format from an API ID string.
// Returns the format portion (e.g. "proto") if the argument parses as
// an API ID, or empty string otherwise.
func ResolveAPIFormat(arg string) string {
	api, err := ParseAPIID(arg)
	if err != nil {
		return ""
	}
	return api.Format
}
