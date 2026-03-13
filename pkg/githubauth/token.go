package githubauth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Token represents a cached GitHub OAuth token.
type Token struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type,omitempty"`
	Scope       string    `json:"scope,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ConfigDir is the directory where apx stores cached credentials.
// Override in tests to use a temp dir.
var ConfigDir = configDirReal

func configDirReal() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "apx")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return dir, nil
}

// TokenPath returns the file path for the cached token for a given org.
// The file name matches the GitHub App name convention: apx-{org}-user-token.json
func TokenPath(org string) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	newPath := filepath.Join(dir, fmt.Sprintf("apx-%s-user-token.json", org))

	// Migrate: if old-style token exists and new doesn't, rename it.
	oldPath := filepath.Join(dir, org+"-github-token.json")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		if _, err := os.Stat(oldPath); err == nil {
			_ = os.Rename(oldPath, newPath)
		}
	}

	return newPath, nil
}

// LoadToken reads a cached token from disk. Returns nil, nil if the file
// does not exist.
func LoadToken(org string) (*Token, error) {
	p, err := TokenPath(org)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read token: %w", err)
	}
	var tok Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, nil
	}
	return &tok, nil
}

// SaveToken persists a token to disk (mode 0600).
func SaveToken(org string, tok *Token) error {
	p, err := TokenPath(org)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		return fmt.Errorf("failed to write token: %w", err)
	}
	return nil
}

// ClearToken removes the cached token for an org.
func ClearToken(org string) error {
	p, err := TokenPath(org)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token: %w", err)
	}
	return nil
}

// --- Generic cache helpers (used by setup.go for app-id, slug, etc.) ---

// ReadCache reads a single-value cache file: ~/.config/apx/{org}-{suffix}.
func ReadCache(org, suffix string) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, org+"-"+suffix))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// WriteCache writes a single-value cache file: ~/.config/apx/{org}-{suffix}.
func WriteCache(org, suffix, value string) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, org+"-"+suffix), []byte(value), 0600)
}
