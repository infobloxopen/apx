package githubauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenExpired(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name string
		tok  Token
		want bool
	}{
		{"fresh expiring token", Token{CreatedAt: now, ExpiresIn: 28800}, false},
		{"past expiry", Token{CreatedAt: now.Add(-9 * time.Hour), ExpiresIn: 28800}, true},
		{"inside skew window", Token{CreatedAt: now.Add(-28800*time.Second + 30*time.Second), ExpiresIn: 28800}, true},
		{"no expiry metadata (legacy)", Token{CreatedAt: now.Add(-1000 * time.Hour)}, false},
		{"expiry but zero created_at", Token{ExpiresIn: 28800}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tok.Expired())
		})
	}
}

func TestTokenRefreshUsable(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name string
		tok  Token
		want bool
	}{
		{"no refresh token", Token{CreatedAt: now}, false},
		{"refresh token, no expiry", Token{CreatedAt: now, RefreshToken: "ghr_x"}, true},
		{"refresh token within lifetime", Token{CreatedAt: now, RefreshToken: "ghr_x", RefreshTokenExpiresIn: 15724800}, true},
		{"refresh token past lifetime", Token{CreatedAt: now.Add(-200 * 24 * time.Hour), RefreshToken: "ghr_x", RefreshTokenExpiresIn: 15724800}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tok.RefreshUsable())
		})
	}
}

func TestTokenFromEnv(t *testing.T) {
	for _, name := range []string{"APX_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(name, "")
	}
	assert.Equal(t, "", TokenFromEnv())

	t.Setenv("GITHUB_TOKEN", "gho_github")
	assert.Equal(t, "gho_github", TokenFromEnv())

	t.Setenv("GH_TOKEN", "gho_gh")
	assert.Equal(t, "gho_gh", TokenFromEnv(), "GH_TOKEN takes precedence over GITHUB_TOKEN")

	t.Setenv("APX_GITHUB_TOKEN", "gho_apx")
	assert.Equal(t, "gho_apx", TokenFromEnv(), "APX_GITHUB_TOKEN takes precedence over all")
}

func TestValidateToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Authorization") {
		case "Bearer good":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	orig := APIBaseURL
	APIBaseURL = srv.URL
	defer func() { APIBaseURL = orig }()

	assert.True(t, ValidateToken("good"))
	assert.False(t, ValidateToken("dead"))
}

func TestEnsureToken_EnvOverride(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	t.Setenv("APX_GITHUB_TOKEN", "gho_from_env")

	got, err := EnsureToken("someorg")
	require.NoError(t, err)
	assert.Equal(t, "gho_from_env", got, "env token bypasses cache and device flow")
}

func TestEnsureToken_ValidCachedToken(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()
	t.Setenv("APX_GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	// Fresh expiring token: trusted by clock, no API call needed.
	require.NoError(t, SaveToken("myorg", &Token{
		AccessToken: "ghu_fresh",
		CreatedAt:   time.Now().UTC(),
		ExpiresIn:   28800,
	}))

	got, err := EnsureToken("myorg")
	require.NoError(t, err)
	assert.Equal(t, "ghu_fresh", got)
}

func TestEnsureToken_LegacyTokenRejectedByAPI(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()
	t.Setenv("APX_GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	origAPI := APIBaseURL
	APIBaseURL = srv.URL
	defer func() { APIBaseURL = origAPI }()

	// Legacy cache file: no expiry metadata, dead at GitHub, no refresh token,
	// and no client-id cached — so the device flow cannot start.
	require.NoError(t, SaveToken("myorg", &Token{
		AccessToken: "ghu_dead",
		CreatedAt:   time.Now().Add(-60 * 24 * time.Hour).UTC(),
	}))

	_, err := EnsureToken("myorg")
	require.Error(t, err, "dead token must not be returned")
	assert.Contains(t, err.Error(), "no GitHub user app configured")

	// The dead token must have been evicted from the cache.
	tok, lerr := LoadToken("myorg")
	require.NoError(t, lerr)
	assert.Nil(t, tok, "dead cached token should be cleared")
}

func TestEnsureToken_RefreshesExpiredToken(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()
	t.Setenv("APX_GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "ghr_refresh", r.Form.Get("refresh_token"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "ghu_renewed",
			"token_type": "bearer",
			"expires_in": 28800,
			"refresh_token": "ghr_next",
			"refresh_token_expires_in": 15724800
		}`))
	}))
	defer srv.Close()
	origGH := GitHubBaseURL
	GitHubBaseURL = srv.URL
	defer func() { GitHubBaseURL = origGH }()

	require.NoError(t, WriteCache("myorg", "user-app-client-id", "Iv1.testclient"))
	require.NoError(t, SaveToken("myorg", &Token{
		AccessToken:           "ghu_expired",
		CreatedAt:             time.Now().Add(-10 * time.Hour).UTC(),
		ExpiresIn:             28800,
		RefreshToken:          "ghr_refresh",
		RefreshTokenExpiresIn: 15724800,
	}))

	got, err := EnsureToken("myorg")
	require.NoError(t, err)
	assert.Equal(t, "ghu_renewed", got)

	// The renewed token (with its new refresh token) must be persisted.
	tok, err := LoadToken("myorg")
	require.NoError(t, err)
	require.NotNil(t, tok)
	assert.Equal(t, "ghu_renewed", tok.AccessToken)
	assert.Equal(t, "ghr_next", tok.RefreshToken)
	assert.False(t, tok.CreatedAt.IsZero(), "refreshed token must carry created_at")
}
