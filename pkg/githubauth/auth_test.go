package githubauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceFlowLogin(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/device/code":
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"device_code":      "dc-123",
				"user_code":        "ABCD-1234",
				"verification_uri": "https://github.com/login/device",
				"expires_in":       900,
				"interval":         1, // 1 second for fast tests
			})

		case "/login/oauth/access_token":
			assert.Equal(t, "POST", r.Method)
			callCount++
			w.Header().Set("Content-Type", "application/json")
			if callCount < 2 {
				// First poll: pending.
				json.NewEncoder(w).Encode(map[string]string{
					"error": "authorization_pending",
				})
			} else {
				// Second poll: success.
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "ghu_test_token_123",
					"token_type":   "bearer",
					"scope":        "repo",
				})
			}

		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer server.Close()

	orig := GitHubBaseURL
	GitHubBaseURL = server.URL
	defer func() { GitHubBaseURL = orig }()

	tok, err := DeviceFlowLogin("test-client-id")
	require.NoError(t, err)
	assert.Equal(t, "ghu_test_token_123", tok.AccessToken)
	assert.Equal(t, "bearer", tok.TokenType)
	assert.Equal(t, "repo", tok.Scope)
	assert.False(t, tok.CreatedAt.IsZero())
	assert.GreaterOrEqual(t, callCount, 2, "should have polled at least twice")
}

func TestDeviceFlowLogin_EmptyClientID(t *testing.T) {
	_, err := DeviceFlowLogin("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client_id is required")
}

func TestDeviceFlowLogin_AccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login/device/code":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"device_code":      "dc-123",
				"user_code":        "ABCD-1234",
				"verification_uri": "https://github.com/login/device",
				"expires_in":       900,
				"interval":         1,
			})
		case "/login/oauth/access_token":
			json.NewEncoder(w).Encode(map[string]string{
				"error": "access_denied",
			})
		}
	}))
	defer server.Close()

	orig := GitHubBaseURL
	GitHubBaseURL = server.URL
	defer func() { GitHubBaseURL = orig }()

	_, err := DeviceFlowLogin("test-client-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "denied")
}

func TestDetectOrg_ParsesHTTPS(t *testing.T) {
	m := gitRemoteRe.FindStringSubmatch("https://github.com/myorg/myrepo.git")
	require.Len(t, m, 2)
	assert.Equal(t, "myorg", m[1])
}

func TestDetectOrg_ParsesSSH(t *testing.T) {
	m := gitRemoteRe.FindStringSubmatch("git@github.com:myorg/myrepo.git")
	require.Len(t, m, 2)
	assert.Equal(t, "myorg", m[1])
}
