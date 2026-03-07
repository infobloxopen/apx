// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package gitea

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Instance represents a running Gitea instance in k3d
type Instance struct {
	URL           string
	AdminUser     string
	AdminPassword string
	AdminToken    string
	Namespace     string
	ClusterName   string
}

// Deploy deploys Gitea to a k3d cluster
func Deploy(ctx context.Context, clusterName, namespace string) (*Instance, error) {
	// Generate Gitea manifest
	config := DefaultGiteaConfig()
	manifest := config.GenerateGiteaManifest(namespace) // Write manifest to temporary file
	tmpDir := os.TempDir()
	manifestPath := filepath.Join(tmpDir, fmt.Sprintf("gitea-%s.yaml", namespace))
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		return nil, fmt.Errorf("failed to write Gitea manifest: %w", err)
	}
	defer os.Remove(manifestPath)

	// Apply manifest to cluster
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", manifestPath,
		"--context", fmt.Sprintf("k3d-%s", clusterName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to deploy Gitea: %w\nOutput: %s", err, output)
	}

	instance := &Instance{
		URL:           fmt.Sprintf("http://localhost:%d", config.Port),
		AdminUser:     config.AdminUser,
		AdminPassword: config.AdminPassword,
		Namespace:     namespace,
		ClusterName:   clusterName,
	}

	return instance, nil
}

// WaitForReady waits for Gitea to be ready to accept requests
func (i *Instance) WaitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	client := &http.Client{Timeout: 3 * time.Second}
	versionURL := fmt.Sprintf("%s/api/v1/version", i.URL)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Gitea to be ready: %w", ctx.Err())
		case <-ticker.C:
			resp, err := client.Get(versionURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil // Gitea is ready
			}
			if resp != nil {
				resp.Body.Close()
			}
			// Continue waiting
		}
	}
}

// CreateAdminToken creates an API token for the admin user
func (i *Instance) CreateAdminToken(ctx context.Context) error {
	// Wait for Gitea to complete initial setup
	time.Sleep(5 * time.Second)

	// Use Gitea CLI to create admin user and token
	// This is done via kubectl exec into the Gitea pod

	podName, err := i.getGiteaPodName(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Gitea pod name: %w", err)
	}

	// Create admin user using Gitea CLI (must run as 'git' user, not root)
	createUserCmd := exec.CommandContext(ctx, "kubectl", "exec", "-n", i.Namespace,
		podName, "--context", fmt.Sprintf("k3d-%s", i.ClusterName), "--",
		"su", "-c",
		fmt.Sprintf("gitea admin user create --username %s --password %s --email admin@example.com --admin --must-change-password=false",
			i.AdminUser, i.AdminPassword),
		"git")
	_ = createUserCmd.Run() // Ignore error if user already exists

	// Generate token using Gitea CLI (must run as 'git' user, not root)
	tokenCmd := exec.CommandContext(ctx, "kubectl", "exec", "-n", i.Namespace,
		podName, "--context", fmt.Sprintf("k3d-%s", i.ClusterName), "--",
		"su", "-c",
		fmt.Sprintf("gitea admin user generate-access-token --username %s --token-name e2e-test-token --scopes write:repository,write:user,write:admin",
			i.AdminUser),
		"git")
	output, err := tokenCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate admin token: %w\nOutput: %s", err, output)
	}

	// Extract token from output
	// Gitea outputs: "Access token was successfully created: <token>"
	// or just the token on a line by itself depending on version
	rawOutput := strings.TrimSpace(string(output))
	token := rawOutput

	// Check for "Access token was successfully created:" prefix
	for _, line := range strings.Split(rawOutput, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Access token was successfully created:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				token = strings.TrimSpace(parts[1])
			}
			break
		}
	}

	if token == "" {
		return fmt.Errorf("empty token from Gitea admin output: %s", rawOutput)
	}

	i.AdminToken = token

	return nil
}

// getGiteaPodName retrieves the name of the Gitea pod
func (i *Instance) getGiteaPodName(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods",
		"-n", i.Namespace,
		"--context", fmt.Sprintf("k3d-%s", i.ClusterName),
		"-l", "app=gitea",
		"-o", "jsonpath={.items[0].metadata.name}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get Gitea pod: %w\nOutput: %s", err, output)
	}

	podName := strings.TrimSpace(string(output))
	if podName == "" {
		return "", fmt.Errorf("no Gitea pod found in namespace %s", i.Namespace)
	}

	return podName, nil
}

// GetLogs retrieves logs from the Gitea container
func (i *Instance) GetLogs(ctx context.Context) (string, error) {
	podName, err := i.getGiteaPodName(ctx)
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, "kubectl", "logs",
		"-n", i.Namespace,
		"--context", fmt.Sprintf("k3d-%s", i.ClusterName),
		podName,
		"--tail=100")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	return string(output), nil
}

// Delete removes the Gitea deployment from the cluster
func (i *Instance) Delete(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "kubectl", "delete", "namespace", i.Namespace,
		"--context", fmt.Sprintf("k3d-%s", i.ClusterName),
		"--wait=false") // Don't wait for full deletion
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete Gitea namespace: %w", err)
	}
	return nil
}

// DefaultGiteaConfig returns default Gitea deployment configuration
type GiteaConfig struct {
	Image         string
	Port          int
	AdminUser     string
	AdminPassword string
}

func DefaultGiteaConfig() *GiteaConfig {
	return &GiteaConfig{
		Image:         "gitea/gitea:1.22",
		Port:          3000,
		AdminUser:     "gitea_admin",
		AdminPassword: "admin123",
	}
}

// GenerateGiteaManifest generates Kubernetes manifest for Gitea deployment
func (c *GiteaConfig) GenerateGiteaManifest(namespace string) string {
	return fmt.Sprintf(`---
apiVersion: v1
kind: Namespace
metadata:
  name: %s
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: gitea-config
  namespace: %s
data:
  app.ini: |
    [database]
    DB_TYPE = sqlite3
    PATH = /data/gitea/gitea.db
    
    [repository]
    ROOT = /data/git/repositories
    
    [server]
    PROTOCOL = http
    DOMAIN = localhost
    HTTP_PORT = %d
    ROOT_URL = http://localhost:%d/
    DISABLE_SSH = true
    START_SSH_SERVER = false
    
    [service]
    DISABLE_REGISTRATION = false
    REQUIRE_SIGNIN_VIEW = false
    ENABLE_NOTIFY_MAIL = false
    
    [webhook]
    ALLOWED_HOST_LIST = *
    
    [mailer]
    ENABLED = false
    
    [log]
    MODE = console
    LEVEL = Info
    
    [security]
    INSTALL_LOCK = true
    SECRET_KEY = E2E-TEST-SECRET-KEY-CHANGE-ME
    INTERNAL_TOKEN = E2E-TEST-INTERNAL-TOKEN-CHANGE-ME
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: gitea-data
  namespace: %s
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gitea
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gitea
  template:
    metadata:
      labels:
        app: gitea
    spec:
      securityContext:
        fsGroup: 1000
      initContainers:
      - name: init-config
        image: %s
        command: ['sh', '-c', 'mkdir -p /data/gitea/conf && cp /etc/gitea/app.ini /data/gitea/conf/app.ini && chown -R 1000:1000 /data']
        securityContext:
          runAsUser: 0
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /etc/gitea
      containers:
      - name: gitea
        image: %s
        ports:
        - containerPort: %d
          name: http
        env:
        - name: USER_UID
          value: "1000"
        - name: USER_GID
          value: "1000"
        - name: GITEA__database__DB_TYPE
          value: sqlite3
        - name: GITEA__database__PATH
          value: /data/gitea/gitea.db
        - name: GITEA__security__INSTALL_LOCK
          value: "true"
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /etc/gitea
        readinessProbe:
          httpGet:
            path: /api/v1/version
            port: %d
          initialDelaySeconds: 10
          periodSeconds: 3
          timeoutSeconds: 3
          failureThreshold: 20
        livenessProbe:
          httpGet:
            path: /api/v1/version
            port: %d
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 5
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: gitea-data
      - name: config
        configMap:
          name: gitea-config
---
apiVersion: v1
kind: Service
metadata:
  name: gitea
  namespace: %s
spec:
  type: NodePort
  ports:
  - port: %d
    targetPort: %d
    nodePort: 30000
    protocol: TCP
    name: http
  selector:
    app: gitea
`, namespace, namespace, c.Port, c.Port, namespace, namespace, c.Image, c.Image, c.Port, c.Port, c.Port, namespace, c.Port, c.Port)
}

// HealthCheck performs a health check on the Gitea instance
func (i *Instance) HealthCheck(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/version", i.URL))
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned status %d: %s", resp.StatusCode, body)
	}

	return nil
}
