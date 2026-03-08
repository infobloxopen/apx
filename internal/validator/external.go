package validator

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
)

// ValidateExternalRegistration validates an ExternalRegistration struct.
// It delegates to the registration's own Validate method and performs
// additional cross-cutting checks.
func ValidateExternalRegistration(reg *config.ExternalRegistration) error {
	return reg.Validate()
}

// ValidateImportMode checks if an import mode value is valid.
func ValidateImportMode(mode string) error {
	switch mode {
	case "", config.ImportModePreserve, config.ImportModeRewrite:
		return nil
	default:
		return fmt.Errorf("invalid import_mode %q: must be %q or %q",
			mode, config.ImportModePreserve, config.ImportModeRewrite)
	}
}

// ValidateOrigin checks if an origin value is valid.
func ValidateOrigin(origin string) error {
	switch origin {
	case "", config.OriginExternal, config.OriginForked:
		return nil
	default:
		return fmt.Errorf("invalid origin %q: must be %q or %q",
			origin, config.OriginExternal, config.OriginForked)
	}
}

// ValidateExternalAPIs validates a list of external API registrations for
// internal consistency (no duplicate IDs, no conflicting paths).
func ValidateExternalAPIs(registrations []config.ExternalRegistration) []error {
	var errs []error
	seenIDs := make(map[string]bool)
	seenPaths := make(map[string]string) // path -> ID that owns it

	for i, reg := range registrations {
		if err := reg.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("external_apis[%d]: %w", i, err))
			continue
		}

		if seenIDs[reg.ID] {
			errs = append(errs, fmt.Errorf("external_apis[%d]: duplicate external API ID: %s", i, reg.ID))
		}
		seenIDs[reg.ID] = true

		if ownerID, exists := seenPaths[reg.ManagedPath]; exists {
			errs = append(errs, fmt.Errorf("external_apis[%d]: managed_path %q conflicts with %s",
				i, reg.ManagedPath, ownerID))
		}
		seenPaths[reg.ManagedPath] = reg.ID
	}

	return errs
}

// ValidateRepoURL checks that a repository URL is well-formed.
// Accepts both full URLs (https://github.com/org/repo) and
// shorthand (github.com/org/repo).
func ValidateRepoURL(repoURL string) error {
	if repoURL == "" {
		return fmt.Errorf("URL is empty")
	}

	// Must have at least a host part with a dot
	if strings.Contains(repoURL, "://") {
		// Full URL — basic structure check
		if !strings.Contains(repoURL[strings.Index(repoURL, "://")+3:], "/") {
			return fmt.Errorf("URL must have a path after the host")
		}
		return nil
	}

	// Shorthand: must have host.tld/path
	parts := strings.SplitN(repoURL, "/", 3)
	if len(parts) < 2 {
		return fmt.Errorf("expected host/path format, got %q", repoURL)
	}
	if !strings.Contains(parts[0], ".") {
		return fmt.Errorf("host %q must contain a dot", parts[0])
	}
	return nil
}
