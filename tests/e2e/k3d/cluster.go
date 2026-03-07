// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package k3d

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Cluster represents a k3d cluster
type Cluster struct {
	Name      string
	Port      int
	CreatedAt time.Time
}

// CreateCluster creates a new k3d cluster with Gitea port mapping
func CreateCluster(ctx context.Context, name string, giteaPort int) (*Cluster, error) {
	// Build k3d cluster create command
	// Note: We map host port to NodePort 30000 via the loadbalancer
	args := []string{
		"cluster", "create", name,
		"--port", fmt.Sprintf("%d:30000@loadbalancer", giteaPort), // Map host port to NodePort via LB
		"--wait",
		"--timeout", "2m",
		"--k3s-arg", "--disable=traefik@server:0", // Disable traefik ingress controller
	}

	cmd := exec.CommandContext(ctx, "k3d", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create k3d cluster %s: %w\nOutput: %s", name, err, output)
	}

	cluster := &Cluster{
		Name:      name,
		Port:      giteaPort,
		CreatedAt: time.Now(),
	}

	return cluster, nil
}

// Delete removes the k3d cluster
func (c *Cluster) Delete(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "delete", c.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete k3d cluster %s: %w\nOutput: %s", c.Name, err, output)
	}
	return nil
}

// Exists checks if the cluster exists
func (c *Cluster) Exists(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "list", "-o", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to list k3d clusters: %w\nOutput: %s", err, output)
	}

	// Simple check: if cluster name appears in output, it exists
	return strings.Contains(string(output), c.Name), nil
}

// WaitForReady waits for the cluster to be ready
func (c *Cluster) WaitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for cluster %s to be ready: %w", c.Name, ctx.Err())
		case <-ticker.C:
			// Check if kubectl can connect to the cluster using k3d context
			cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes",
				"--context", fmt.Sprintf("k3d-%s", c.Name))
			if err := cmd.Run(); err == nil {
				return nil // Cluster is ready
			}
			// Continue waiting
		}
	}
}

// GetKubeconfig returns the kubeconfig path for the cluster
func GetKubeconfig(clusterName string) string {
	// k3d writes kubeconfig to default location with cluster-specific context
	// We can use kubectl with KUBECONFIG env var or --context flag
	return "~/.kube/config" // k3d merges into default kubeconfig
}
