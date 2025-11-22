// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package k3d

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CleanupAllE2EClusters removes all k3d clusters with "apx-e2e-" prefix
// This is a safety net for orphaned clusters from failed tests
func CleanupAllE2EClusters(ctx context.Context) error {
	// List all k3d clusters
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "list", "-o", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list k3d clusters: %w\nOutput: %s", err, output)
	}

	// Parse cluster names (simple string search for apx-e2e- prefix)
	lines := strings.Split(string(output), "\n")
	var clustersToDelete []string
	for _, line := range lines {
		if strings.Contains(line, "apx-e2e-") {
			// Extract cluster name (basic parsing)
			// Full JSON parsing would be better, but this works for cleanup
			parts := strings.Split(line, `"`)
			for _, part := range parts {
				if strings.HasPrefix(part, "apx-e2e-") {
					clustersToDelete = append(clustersToDelete, part)
					break
				}
			}
		}
	}

	// Delete each cluster
	for _, clusterName := range clustersToDelete {
		cmd := exec.CommandContext(ctx, "k3d", "cluster", "delete", clusterName)
		if err := cmd.Run(); err != nil {
			// Log but don't fail - best effort cleanup
			fmt.Printf("Warning: failed to delete cluster %s: %v\n", clusterName, err)
		}
	}

	return nil
}

// CleanupOrphanedContainers removes Docker containers related to E2E tests
func CleanupOrphanedContainers(ctx context.Context) error {
	// Remove containers with "k3d-apx-e2e-" prefix
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=k3d-apx-e2e-", "-q")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list Docker containers: %w", err)
	}

	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return nil // No containers to clean up
	}

	// Remove containers
	args := append([]string{"rm", "-f"}, containerIDs...)
	cmd = exec.CommandContext(ctx, "docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove containers: %w", err)
	}

	return nil
}

// CleanupOrphanedVolumes removes Docker volumes related to E2E tests
func CleanupOrphanedVolumes(ctx context.Context) error {
	// List volumes with "k3d-apx-e2e-" prefix
	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "--filter", "name=k3d-apx-e2e-", "-q")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list Docker volumes: %w", err)
	}

	volumeNames := strings.Fields(string(output))
	if len(volumeNames) == 0 {
		return nil // No volumes to clean up
	}

	// Remove volumes
	args := append([]string{"volume", "rm", "-f"}, volumeNames...)
	cmd = exec.CommandContext(ctx, "docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove volumes: %w", err)
	}

	return nil
}

// CleanupAll performs comprehensive cleanup of all E2E test resources
func CleanupAll(ctx context.Context) error {
	// Clean up in order: clusters (which removes containers/volumes), then orphaned resources
	if err := CleanupAllE2EClusters(ctx); err != nil {
		return fmt.Errorf("cluster cleanup failed: %w", err)
	}

	if err := CleanupOrphanedContainers(ctx); err != nil {
		return fmt.Errorf("container cleanup failed: %w", err)
	}

	if err := CleanupOrphanedVolumes(ctx); err != nil {
		return fmt.Errorf("volume cleanup failed: %w", err)
	}

	return nil
}
