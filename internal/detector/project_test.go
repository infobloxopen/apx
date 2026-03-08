package detector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitRemoteURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantOrg string
		wantRep string
		wantErr bool
	}{
		{name: "SSH shorthand with .git", url: "git@github.com:Infoblox-CTO/apis.git", wantOrg: "Infoblox-CTO", wantRep: "apis"},
		{name: "SSH shorthand without .git", url: "git@github.com:Infoblox-CTO/apis", wantOrg: "Infoblox-CTO", wantRep: "apis"},
		{name: "HTTPS with .git", url: "https://github.com/infobloxopen/apx.git", wantOrg: "infobloxopen", wantRep: "apx"},
		{name: "HTTPS without .git", url: "https://github.com/infobloxopen/apx", wantOrg: "infobloxopen", wantRep: "apx"},
		{name: "SSH URL with scheme", url: "ssh://git@github.com/Infoblox-CTO/apis.git", wantOrg: "Infoblox-CTO", wantRep: "apis"},
		{name: "git protocol URL", url: "git://github.com/myorg/myrepo.git", wantOrg: "myorg", wantRep: "myrepo"},
		{name: "whitespace trimmed", url: "  git@github.com:Infoblox-CTO/apis.git\n", wantOrg: "Infoblox-CTO", wantRep: "apis"},
		{name: "empty URL", url: "", wantErr: true},
		{name: "invalid URL", url: "not-a-url", wantErr: true},
		{name: "local path", url: "/path/to/repo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo, err := ParseGitRemoteURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOrg, org)
			assert.Equal(t, tt.wantRep, repo)
		})
	}
}

func TestDetectFromGitRemote(t *testing.T) {
	orig := gitRemoteURL
	t.Cleanup(func() { gitRemoteURL = orig })

	t.Run("successful detection", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			assert.Equal(t, "origin", remote)
			return "git@github.com:Infoblox-CTO/apis.git", nil
		}
		org, repo, err := DetectFromGitRemote("origin")
		require.NoError(t, err)
		assert.Equal(t, "Infoblox-CTO", org)
		assert.Equal(t, "apis", repo)
	})

	t.Run("git command failure", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			return "", fmt.Errorf("not a git repo")
		}
		_, _, err := DetectFromGitRemote("origin")
		assert.Error(t, err)
	})

	t.Run("unparseable URL", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			return "/local/path", nil
		}
		_, _, err := DetectFromGitRemote("origin")
		assert.Error(t, err)
	})
}

func TestGetSmartDefaults(t *testing.T) {
	orig := gitRemoteURL
	t.Cleanup(func() { gitRemoteURL = orig })

	t.Run("uses git remote when available", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			return "git@github.com:Infoblox-CTO/apis.git", nil
		}
		defaults, err := GetSmartDefaults()
		require.NoError(t, err)
		assert.Equal(t, "Infoblox-CTO", defaults.Org)
		assert.Equal(t, "apis", defaults.Repo)
		assert.Empty(t, defaults.UpstreamOrg, "no upstream when origin is not a fork")
	})

	t.Run("falls back when git remote unavailable", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			return "", fmt.Errorf("no remote")
		}
		defaults, err := GetSmartDefaults()
		require.NoError(t, err)
		assert.NotEmpty(t, defaults.Org)
		assert.NotEmpty(t, defaults.Repo)
	})

	t.Run("always populates languages", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			return "", fmt.Errorf("no remote")
		}
		defaults, err := GetSmartDefaults()
		require.NoError(t, err)
		assert.NotEmpty(t, defaults.Languages)
	})

	t.Run("fork detected via upstream remote", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			switch remote {
			case "origin":
				return "git@github.com:dgarcia/apis.git", nil
			case "upstream":
				return "git@github.com:Infoblox-CTO/apis.git", nil
			}
			return "", fmt.Errorf("unknown remote %s", remote)
		}
		defaults, err := GetSmartDefaults()
		require.NoError(t, err)
		assert.Equal(t, "Infoblox-CTO", defaults.Org, "should use upstream org for consumption")
		assert.Equal(t, "apis", defaults.Repo)
		assert.Equal(t, "Infoblox-CTO", defaults.UpstreamOrg, "should record upstream org")
	})

	t.Run("fork with different repo name upstream", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			switch remote {
			case "origin":
				return "git@github.com:dgarcia/my-fork.git", nil
			case "upstream":
				return "https://github.com/Infoblox-CTO/apis.git", nil
			}
			return "", fmt.Errorf("unknown remote %s", remote)
		}
		defaults, err := GetSmartDefaults()
		require.NoError(t, err)
		assert.Equal(t, "Infoblox-CTO", defaults.Org)
		assert.Equal(t, "apis", defaults.Repo, "should use upstream repo name")
		assert.Equal(t, "Infoblox-CTO", defaults.UpstreamOrg)
	})

	t.Run("no upstream remote means not a fork", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			if remote == "origin" {
				return "git@github.com:dgarcia/apis.git", nil
			}
			return "", fmt.Errorf("no such remote: %s", remote)
		}
		defaults, err := GetSmartDefaults()
		require.NoError(t, err)
		assert.Equal(t, "dgarcia", defaults.Org, "no upstream so use origin org")
		assert.Empty(t, defaults.UpstreamOrg)
	})

	t.Run("upstream same org as origin is not a fork", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			// Both remotes point to the same org (e.g. multiple remotes for same org)
			return "git@github.com:Infoblox-CTO/apis.git", nil
		}
		defaults, err := GetSmartDefaults()
		require.NoError(t, err)
		assert.Equal(t, "Infoblox-CTO", defaults.Org)
		assert.Empty(t, defaults.UpstreamOrg, "same org means not a fork")
	})
}

func TestDetectOrgFromGit(t *testing.T) {
	orig := gitRemoteURL
	t.Cleanup(func() { gitRemoteURL = orig })

	t.Run("prefers git remote over path", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			return "https://github.com/MyOrg/my-repo.git", nil
		}
		org, err := DetectOrgFromGit()
		require.NoError(t, err)
		assert.Equal(t, "MyOrg", org)
	})

	t.Run("prefers upstream over origin", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			switch remote {
			case "upstream":
				return "https://github.com/Infoblox-CTO/apis.git", nil
			case "origin":
				return "https://github.com/dgarcia/apis.git", nil
			}
			return "", fmt.Errorf("no remote")
		}
		org, err := DetectOrgFromGit()
		require.NoError(t, err)
		assert.Equal(t, "Infoblox-CTO", org, "should prefer upstream org")
	})
}

func TestDetectRepoName(t *testing.T) {
	orig := gitRemoteURL
	t.Cleanup(func() { gitRemoteURL = orig })

	t.Run("uses git remote repo name", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			return "git@github.com:Infoblox-CTO/my-apis.git", nil
		}
		repo, err := DetectRepoName()
		require.NoError(t, err)
		assert.Equal(t, "my-apis", repo)
	})

	t.Run("falls back to directory name", func(t *testing.T) {
		gitRemoteURL = func(remote string) (string, error) {
			return "", fmt.Errorf("no remote")
		}
		repo, err := DetectRepoName()
		require.NoError(t, err)
		assert.NotEmpty(t, repo)
	})
}
