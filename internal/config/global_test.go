package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTempHome points os.UserHomeDir at a temp dir for the test. Windows
// resolves the home directory from USERPROFILE, not HOME — overriding the
// wrong one silently leaks the real home into the test.
func setTempHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	} else {
		t.Setenv("HOME", tmp)
	}
	return tmp
}

func TestLoadGlobal_MissingFile(t *testing.T) {
	// Override home so GlobalConfigPath points to a temp dir
	setTempHome(t)

	cfg, err := LoadGlobal()
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
	assert.Empty(t, cfg.Orgs)
}

func TestSaveAndLoadGlobal(t *testing.T) {
	setTempHome(t)

	cfg := &GlobalConfig{
		Version:    1,
		DefaultOrg: "acme",
		Orgs: []KnownOrg{
			{Name: "acme", Repos: []string{"apis"}},
		},
	}
	require.NoError(t, SaveGlobal(cfg))

	loaded, err := LoadGlobal()
	require.NoError(t, err)
	assert.Equal(t, "acme", loaded.DefaultOrg)
	require.Len(t, loaded.Orgs, 1)
	assert.Equal(t, "acme", loaded.Orgs[0].Name)
	assert.Equal(t, []string{"apis"}, loaded.Orgs[0].Repos)
}

func TestAddOrg_NewOrg(t *testing.T) {
	cfg := &GlobalConfig{Version: 1}
	cfg.AddOrg("acme", []string{"apis"})

	require.Len(t, cfg.Orgs, 1)
	assert.Equal(t, "acme", cfg.Orgs[0].Name)
	assert.Equal(t, []string{"apis"}, cfg.Orgs[0].Repos)
}

func TestAddOrg_MergesRepos(t *testing.T) {
	cfg := &GlobalConfig{
		Version: 1,
		Orgs:    []KnownOrg{{Name: "acme", Repos: []string{"apis"}}},
	}
	cfg.AddOrg("acme", []string{"apis", "shared-schemas"})

	require.Len(t, cfg.Orgs, 1)
	assert.Equal(t, []string{"apis", "shared-schemas"}, cfg.Orgs[0].Repos)
}

func TestAddOrg_Idempotent(t *testing.T) {
	cfg := &GlobalConfig{Version: 1}
	cfg.AddOrg("acme", []string{"apis"})
	cfg.AddOrg("acme", []string{"apis"})

	require.Len(t, cfg.Orgs, 1)
	assert.Equal(t, []string{"apis"}, cfg.Orgs[0].Repos)
}

func TestSetDefaultOrg(t *testing.T) {
	cfg := &GlobalConfig{Version: 1}
	cfg.AddOrg("acme", []string{"apis"})
	cfg.SetDefaultOrg("acme")

	assert.Equal(t, "acme", cfg.DefaultOrg)
}

func TestSetDefaultOrg_AddsIfMissing(t *testing.T) {
	cfg := &GlobalConfig{Version: 1}
	cfg.SetDefaultOrg("acme")

	assert.Equal(t, "acme", cfg.DefaultOrg)
	require.Len(t, cfg.Orgs, 1)
	assert.Equal(t, "acme", cfg.Orgs[0].Name)
}

func TestKnownOrgNames(t *testing.T) {
	cfg := &GlobalConfig{
		Version: 1,
		Orgs: []KnownOrg{
			{Name: "acme"},
			{Name: "bigcorp"},
		},
	}
	assert.Equal(t, []string{"acme", "bigcorp"}, cfg.KnownOrgNames())
}

func TestFindOrg(t *testing.T) {
	cfg := &GlobalConfig{
		Version: 1,
		Orgs:    []KnownOrg{{Name: "acme", Repos: []string{"apis"}}},
	}

	found := cfg.FindOrg("acme")
	require.NotNil(t, found)
	assert.Equal(t, "acme", found.Name)

	assert.Nil(t, cfg.FindOrg("unknown"))
}

func TestGlobalConfigPath(t *testing.T) {
	tmp := setTempHome(t)

	p, err := GlobalConfigPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, ".config", "apx", "config.yaml"), p)

	// Directory should have been created
	_, err = os.Stat(filepath.Join(tmp, ".config", "apx"))
	assert.NoError(t, err)
}
