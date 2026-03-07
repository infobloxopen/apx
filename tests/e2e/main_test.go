// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/infobloxopen/apx/tests/e2e/gitea"
	"github.com/infobloxopen/apx/tests/e2e/k3d"
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
	if os.Getenv("E2E_SKIP") != "" {
		t.Skip("Skipping E2E tests (E2E_SKIP is set)")
	}

	// Skip if k3d is not installed (e.g., regular CI test run without E2E deps)
	if _, err := exec.LookPath("k3d"); err != nil {
		t.Skip("Skipping E2E tests: k3d not found in PATH (install with: make install-e2e-deps)")
	}

	// Skip if Docker is not running
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("Skipping E2E tests: Docker is not running")
	}

	// Track total execution time for SC-001 (<5 minute requirement)
	suiteStart := time.Now()

	// Setup E2E environment (k3d cluster + Gitea)
	env := setupE2EEnvironment(t)

	// Ensure cleanup happens even on test failure
	t.Cleanup(func() {
		cleanupE2EEnvironment(t, env)

		elapsed := time.Since(suiteStart)
		t.Logf("E2E suite total execution time: %s", elapsed.Round(time.Second))
		if elapsed > 5*time.Minute {
			t.Errorf("SC-001 VIOLATION: E2E suite took %s (limit: 5 minutes)", elapsed.Round(time.Second))
		}
	})

	// Run testscript scenarios
	testscript.Run(t, testscript.Params{
		Dir: "../../testdata/script/e2e",
		Setup: func(e *testscript.Env) error {
			// Setup apx binary in the testscript workspace
			if err := setupAPXBinary(e); err != nil {
				return fmt.Errorf("failed to setup apx binary: %w", err)
			}

			// Inject Gitea URL and token into testscript environment
			e.Setenv("GITEA_URL", env.GiteaURL)
			e.Setenv("GITEA_TOKEN", env.AdminToken)
			e.Setenv("E2E_CLUSTER", env.ClusterID)

			// Set test-specific timestamp for unique repository names
			e.Setenv("TIMESTAMP", fmt.Sprintf("%d", time.Now().Unix()))

			// Set testing environment variables
			e.Setenv("APX_DISABLE_TTY", "1")
			e.Setenv("NO_COLOR", "1")
			e.Setenv("CI", "1")

			return nil
		},
		RequireExplicitExec: true,
	})
}

// E2EEnvironment holds the state of the E2E test environment
type E2EEnvironment struct {
	ClusterID   string          // k3d cluster name
	GiteaURL    string          // Gitea instance URL (e.g., http://localhost:3000)
	AdminToken  string          // Gitea admin API token
	GiteaPort   int             // Host port mapped to Gitea
	Cluster     *k3d.Cluster    // k3d cluster instance
	Gitea       *gitea.Instance // Gitea instance
	GiteaClient *gitea.Client   // Gitea API client
}

// setupE2EEnvironment creates k3d cluster and deploys Gitea
func setupE2EEnvironment(t *testing.T) *E2EEnvironment {
	t.Helper()

	ctx := context.Background()

	// Check if we're in debug mode (keeps environment running after test)
	debugMode := os.Getenv("E2E_DEBUG") != ""
	if debugMode {
		t.Log("E2E_DEBUG=1 detected - cluster will remain running after test")
	}

	// Generate unique cluster name with timestamp
	clusterName := fmt.Sprintf("apx-e2e-%d", time.Now().Unix())
	giteaPort := 3000 // Fixed port for now, can be made dynamic if needed
	namespace := "gitea-e2e"

	t.Logf("Creating k3d cluster: %s", clusterName)

	// 1. Create k3d cluster
	cluster, err := k3d.CreateCluster(ctx, clusterName, giteaPort)
	if err != nil {
		t.Fatalf("Failed to create k3d cluster: %v\n\nHow to fix:\n  1. Ensure Docker is running: docker info\n  2. Check port %d is free: lsof -i :%d\n  3. Clean stale clusters: make clean-e2e\n  4. Check Docker resources: need ~2GB RAM", err, giteaPort, giteaPort)
	}

	// Wait for cluster to be ready
	t.Log("Waiting for k3d cluster to be ready...")
	if err := cluster.WaitForReady(ctx, 2*time.Minute); err != nil {
		t.Fatalf("Cluster failed to become ready: %v", err)
	}
	t.Log("k3d cluster is ready")

	// 2. Deploy Gitea to cluster
	t.Log("Deploying Gitea to cluster...")
	giteaInstance, err := gitea.Deploy(ctx, clusterName, namespace)
	if err != nil {
		t.Fatalf("Failed to deploy Gitea: %v", err)
	}

	// 3. Wait for Gitea readiness
	t.Log("Waiting for Gitea to be ready...")
	if err := giteaInstance.WaitForReady(ctx, 3*time.Minute); err != nil {
		// Try to get logs for debugging
		if logs, logErr := giteaInstance.GetLogs(ctx); logErr == nil {
			t.Logf("Gitea pod logs:\n%s", logs)
		}
		t.Fatalf("Gitea failed to become ready: %v\n\nHow to fix:\n  1. Check Docker has enough resources (~2GB RAM)\n  2. Inspect pod: kubectl --kubeconfig=/tmp/k3d-kubeconfig-* get pods -n gitea-e2e\n  3. View logs: kubectl --kubeconfig=/tmp/k3d-kubeconfig-* logs -n gitea-e2e -l app=gitea\n  4. Try cleanup and retry: make clean-e2e && make test-e2e", err)
	}
	t.Log("Gitea is ready")

	// 4. Create admin token
	t.Log("Creating admin user and token...")
	if err := giteaInstance.CreateAdminToken(ctx); err != nil {
		t.Fatalf("Failed to create admin token: %v\n\nHow to fix:\n  1. Check Gitea is responsive: curl %s/api/v1/version\n  2. View Gitea logs for auth errors\n  3. Try cleanup and retry: make clean-e2e && make test-e2e", err, giteaInstance.URL)
	}
	t.Logf("Admin token created: %s...", giteaInstance.AdminToken[:8])

	// 5. Create Gitea API client
	giteaClient := gitea.NewClient(giteaInstance.URL, giteaInstance.AdminToken)

	// Verify connection
	t.Log("Verifying Gitea API connection...")
	if err := giteaInstance.HealthCheck(ctx); err != nil {
		t.Fatalf("Gitea health check failed: %v", err)
	}

	env := &E2EEnvironment{
		ClusterID:   clusterName,
		GiteaURL:    giteaInstance.URL,
		AdminToken:  giteaInstance.AdminToken,
		GiteaPort:   giteaPort,
		Cluster:     cluster,
		Gitea:       giteaInstance,
		GiteaClient: giteaClient,
	}

	t.Logf("E2E environment ready: cluster=%s, gitea=%s", env.ClusterID, env.GiteaURL)
	return env
}

// cleanupE2EEnvironment tears down k3d cluster and cleans up resources
func cleanupE2EEnvironment(t *testing.T, env *E2EEnvironment) {
	t.Helper()

	ctx := context.Background()

	// Skip cleanup in debug mode
	if os.Getenv("E2E_DEBUG") != "" {
		t.Logf("E2E_DEBUG=1 - skipping cleanup. Manual cleanup: make clean-e2e")
		t.Logf("Gitea URL: %s", env.GiteaURL)
		t.Logf("Admin token: %s", env.AdminToken)
		t.Logf("Cluster: %s", env.ClusterID)
		return
	}

	t.Logf("Cleaning up E2E environment: cluster=%s", env.ClusterID)

	// 1. Delete test repositories via Gitea API (best effort)
	if env.GiteaClient != nil {
		repos, err := env.GiteaClient.ListRepositories(ctx)
		if err == nil {
			for _, repo := range repos {
				t.Logf("Deleting repository: %s", repo.FullName)
				_ = env.GiteaClient.DeleteRepository(ctx, repo.Owner.Username, repo.Name)
			}
		}
	}

	// 2. Stop Gitea (delete namespace)
	if env.Gitea != nil {
		t.Log("Deleting Gitea deployment...")
		if err := env.Gitea.Delete(ctx); err != nil {
			t.Logf("Warning: failed to delete Gitea: %v", err)
		}
	}

	// 3. Delete k3d cluster
	if env.Cluster != nil {
		t.Logf("Deleting k3d cluster: %s", env.Cluster.Name)
		if err := env.Cluster.Delete(ctx); err != nil {
			t.Logf("Warning: failed to delete cluster: %v", err)
		}
	}

	t.Log("E2E environment cleanup complete")
}

// setupAPXBinary builds or copies the apx binary into the testscript workspace
func setupAPXBinary(env *testscript.Env) error {
	// Create bin directory in the test workspace
	binDir := filepath.Join(env.WorkDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	binaryName := "apx"
	if runtime.GOOS == "windows" {
		binaryName = "apx.exe"
	}

	// Check if the binary already exists in ./bin/ (built by CI or make)
	apxBinaryPath := filepath.Join("..", "..", "bin", binaryName)
	destPath := filepath.Join(binDir, binaryName)

	if _, err := os.Stat(apxBinaryPath); err == nil {
		// Copy the pre-built binary to the test workspace
		if err := copyBinaryFile(apxBinaryPath, destPath); err != nil {
			return err
		}
	} else {
		// Build the binary fresh
		cmd := exec.Command("go", "build", "-o", destPath, "../../cmd/apx")
		cmd.Env = os.Environ()
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to build apx binary: %w\nOutput: %s", err, output)
		}
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return err
	}

	// Add the bin directory to PATH
	newPath := binDir + string(os.PathListSeparator) + env.Getenv("PATH")
	env.Setenv("PATH", newPath)

	return nil
}

// copyBinaryFile copies a file from src to dst
func copyBinaryFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	data, err := io.ReadAll(srcFile)
	if err != nil {
		return err
	}
	srcFile.Close()

	return os.WriteFile(dst, data, 0755)
}
