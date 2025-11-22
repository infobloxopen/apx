// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package testhelpers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/tests/e2e/gitea"
) // AssertFileExists checks if a file exists at the given path
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist: %s", path)
	}
}

// AssertFileContains checks if a file contains the given substring
func AssertFileContains(t *testing.T, path, substring string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if !strings.Contains(string(content), substring) {
		t.Fatalf("File %s does not contain expected substring: %s", path, substring)
	}
}

// AssertGitHistory checks if git history contains expected commits
func AssertGitHistory(t *testing.T, repo *GitRepo, expectedMessages []string) {
	t.Helper()

	log, err := repo.GetLog("%s", len(expectedMessages))
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}

	logLines := strings.Split(strings.TrimSpace(log), "\n")
	if len(logLines) < len(expectedMessages) {
		t.Fatalf("Expected at least %d commits, got %d", len(expectedMessages), len(logLines))
	}

	for i, expectedMsg := range expectedMessages {
		if !strings.Contains(logLines[i], expectedMsg) {
			t.Fatalf("Commit %d: expected message to contain %q, got %q",
				i, expectedMsg, logLines[i])
		}
	}
}

// AssertPRExists checks if a pull request exists with the given title
func AssertPRExists(t *testing.T, client *gitea.Client, owner, repo, title string) *gitea.PullRequest {
	t.Helper()

	ctx := context.Background()
	prs, err := client.ListPullRequests(ctx, owner, repo, "all")
	if err != nil {
		t.Fatalf("Failed to list pull requests: %v", err)
	}

	for _, pr := range prs {
		if pr.Title == title {
			return pr
		}
	}

	t.Fatalf("Pull request with title %q not found", title)
	return nil
}

// AssertPRState checks if a pull request has the expected state
func AssertPRState(t *testing.T, pr *gitea.PullRequest, expectedState string) {
	t.Helper()

	if pr.State != expectedState {
		t.Fatalf("Expected PR state to be %q, got %q", expectedState, pr.State)
	}
}

// AssertTagExists checks if a git tag exists in the repository
func AssertTagExists(t *testing.T, client *gitea.Client, owner, repo, tagName string) {
	t.Helper()

	ctx := context.Background()
	tags, err := client.ListTags(ctx, owner, repo)
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}

	for _, tag := range tags {
		if tag.Name == tagName {
			return
		}
	}

	t.Fatalf("Tag %q not found", tagName)
}

// AssertDirectoryStructure checks if expected files/directories exist
func AssertDirectoryStructure(t *testing.T, basePath string, expectedPaths []string) {
	t.Helper()

	for _, relPath := range expectedPaths {
		fullPath := filepath.Join(basePath, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Fatalf("Expected path does not exist: %s", fullPath)
		}
	}
}

// AssertNoError is a helper to check errors in tests
func AssertNoError(t *testing.T, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", context, err)
	}
}

// AssertError is a helper to check that an error occurred
func AssertError(t *testing.T, err error, context string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error but got nil", context)
	}
}

// AssertStringContains checks if a string contains a substring
func AssertStringContains(t *testing.T, str, substring, context string) {
	t.Helper()
	if !strings.Contains(str, substring) {
		t.Fatalf("%s: expected string to contain %q, got %q", context, substring, str)
	}
}
