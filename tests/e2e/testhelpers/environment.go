// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package testhelpers

import (
"fmt"
"os"
"path/filepath"
"testing"
)

// TestEnvironment manages test workspace and cleanup
type TestEnvironment struct {
	t       *testing.T
	WorkDir string
	TempDir string
}

// NewTestEnvironment creates a new test environment with temporary workspace
func NewTestEnvironment(t *testing.T, prefix string) *TestEnvironment {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("apx-e2e-%s-*", prefix))
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	env := &TestEnvironment{
		t:       t,
		WorkDir: tmpDir,
		TempDir: tmpDir,
	}

	// Register cleanup
	t.Cleanup(func() {
		if !t.Failed() || os.Getenv("E2E_KEEP_FAILED") == "" {
			os.RemoveAll(tmpDir)
		} else {
			t.Logf("Test failed - keeping workspace: %s", tmpDir)
		}
	})

	return env
}

// Path returns an absolute path within the test environment
func (e *TestEnvironment) Path(relativePath string) string {
	return filepath.Join(e.WorkDir, relativePath)
}

// MkdirAll creates a directory and all parent directories
func (e *TestEnvironment) MkdirAll(relativePath string) string {
	fullPath := e.Path(relativePath)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		e.t.Fatalf("Failed to create directory %s: %v", fullPath, err)
	}
	return fullPath
}

// WriteFile writes content to a file in the test environment
func (e *TestEnvironment) WriteFile(relativePath, content string) string {
	fullPath := e.Path(relativePath)
	dir := filepath.Dir(fullPath)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		e.t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		e.t.Fatalf("Failed to write file %s: %v", fullPath, err)
	}

	return fullPath
}

// ReadFile reads content from a file in the test environment
func (e *TestEnvironment) ReadFile(relativePath string) string {
	fullPath := e.Path(relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		e.t.Fatalf("Failed to read file %s: %v", fullPath, err)
	}
	return string(content)
}

// FileExists checks if a file exists in the test environment
func (e *TestEnvironment) FileExists(relativePath string) bool {
	_, err := os.Stat(e.Path(relativePath))
	return err == nil
}

// Chdir changes the current working directory to the test environment
func (e *TestEnvironment) Chdir() string {
	if err := os.Chdir(e.WorkDir); err != nil {
		e.t.Fatalf("Failed to change directory: %v", err)
	}
	return e.WorkDir
}
