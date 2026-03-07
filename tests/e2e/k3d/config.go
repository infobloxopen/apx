// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package k3d

import (
	"fmt"
	"time"
)

// ClusterConfig holds configuration for k3d cluster creation
type ClusterConfig struct {
	Name           string
	GiteaPort      int
	WaitTimeout    time.Duration
	DisableTraefik bool
	K3sVersion     string // e.g., "v1.28.3-k3s1"
}

// DefaultClusterConfig returns default configuration for E2E testing
func DefaultClusterConfig(clusterName string) *ClusterConfig {
	return &ClusterConfig{
		Name:           clusterName,
		GiteaPort:      3000, // Default Gitea port
		WaitTimeout:    2 * time.Minute,
		DisableTraefik: true, // Don't need ingress controller for tests
		K3sVersion:     "",   // Use latest stable
	}
}

// WithGiteaPort sets a custom Gitea port
func (c *ClusterConfig) WithGiteaPort(port int) *ClusterConfig {
	c.GiteaPort = port
	return c
}

// WithTimeout sets cluster creation timeout
func (c *ClusterConfig) WithTimeout(timeout time.Duration) *ClusterConfig {
	c.WaitTimeout = timeout
	return c
}

// GiteaDeploymentConfig holds configuration for Gitea deployment
type GiteaDeploymentConfig struct {
	Image         string
	Port          int
	AdminUser     string
	AdminPassword string
	AdminEmail    string
}

// DefaultGiteaConfig returns default Gitea deployment configuration
func DefaultGiteaConfig() *GiteaDeploymentConfig {
	return &GiteaDeploymentConfig{
		Image:         "gitea/gitea:1.22",
		Port:          3000,
		AdminUser:     "gitea_admin",
		AdminPassword: "admin123",
		AdminEmail:    "admin@example.com",
	}
}

// GenerateGiteaManifest generates Kubernetes manifest for Gitea deployment
func (c *GiteaDeploymentConfig) GenerateGiteaManifest(namespace string) string {
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
