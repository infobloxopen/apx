package e2e
// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		// Add custom commands here if needed
	}))
}

// TestE2E runs end-to-end integration tests using testscript
func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	// Check if E2E_SKIP is set

























































































}	t.Logf("E2E environment cleanup (placeholder): cluster=%s", env.ClusterID)		// 4. Verify no orphaned resources remain	// 3. Delete k3d cluster	// 2. Stop Gitea container	// 1. Delete test repositories via Gitea API	// TODO: Phase 2 implementation will add:		}		return		t.Logf("Cluster: %s", env.ClusterID)		t.Logf("Admin token: %s", env.AdminToken)		t.Logf("Gitea URL: %s", env.GiteaURL)		t.Logf("E2E_DEBUG=1 - skipping cleanup. Manual cleanup: make clean-e2e")	if os.Getenv("E2E_DEBUG") != "" {	// Skip cleanup in debug mode		t.Helper()func cleanupE2EEnvironment(t *testing.T, env *E2EEnvironment) {// cleanupE2EEnvironment tears down k3d cluster and cleans up resources}	return env		t.Logf("E2E environment setup (placeholder): cluster=%s, gitea=%s", env.ClusterID, env.GiteaURL)		}		GiteaPort:  3000,                     // Will be allocated dynamically in Phase 2		AdminToken: "placeholder-token",      // Will be generated in Phase 2		GiteaURL:   "http://localhost:3000", // Will be dynamic in Phase 2		ClusterID:  fmt.Sprintf("apx-e2e-%d", time.Now().Unix()),	env := &E2EEnvironment{	// Placeholder for now		// 5. Return environment details	// 4. Create admin token	// 3. Wait for Gitea readiness	// 2. Deploy Gitea to cluster	// 1. Create k3d cluster	// TODO: Phase 2 implementation will add:		}		t.Log("E2E_DEBUG=1 detected - cluster will remain running after test")	if debugMode {		debugMode := os.Getenv("E2E_DEBUG") != ""	// Check if we're in debug mode (keeps environment running after test)		t.Helper()func setupE2EEnvironment(t *testing.T) *E2EEnvironment {// setupE2EEnvironment creates k3d cluster and deploys Gitea}	GiteaPort  int    // Host port mapped to Gitea	AdminToken string // Gitea admin API token	GiteaURL   string // Gitea instance URL (e.g., http://localhost:3000)	ClusterID  string // k3d cluster nametype E2EEnvironment struct {// E2EEnvironment holds the state of the E2E test environment}	})		},			return nil						e.Setenv("TIMESTAMP", fmt.Sprintf("%d", time.Now().Unix()))			// Set test-specific timestamp for unique repository names						e.Setenv("E2E_CLUSTER", env.ClusterID)			e.Setenv("GITEA_TOKEN", env.AdminToken)			e.Setenv("GITEA_URL", env.GiteaURL)			// Inject Gitea URL and token into testscript environment		Setup: func(e *testscript.Env) error {		Dir: "../../testdata/script/e2e",	testscript.Run(t, testscript.Params{	// Run testscript scenarios	})		cleanupE2EEnvironment(t, env)	t.Cleanup(func() {	// Ensure cleanup happens even on test failure		env := setupE2EEnvironment(t)	// Setup E2E environment (k3d cluster + Gitea)	}		t.Skip("Skipping E2E tests (E2E_SKIP is set)")	if os.Getenv("E2E_SKIP") != "" {