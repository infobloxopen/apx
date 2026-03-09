package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffoldPythonPackage(t *testing.T) {
	dir := t.TempDir()

	err := ScaffoldPythonPackage(dir, "acme-payments-ledger-v1", "acme_apis.payments.ledger.v1")
	require.NoError(t, err)

	// pyproject.toml exists and has correct content.
	pyproject, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(pyproject), `name = "acme-payments-ledger-v1"`)
	assert.Contains(t, string(pyproject), `include = ["acme_apis*"]`)
	assert.Contains(t, string(pyproject), `requires-python = ">=3.9"`)

	// Top-level namespace __init__.py uses pkgutil.extend_path.
	nsInit, err := os.ReadFile(filepath.Join(dir, "acme_apis", "__init__.py"))
	require.NoError(t, err)
	assert.Contains(t, string(nsInit), "extend_path")

	// Intermediate packages have __init__.py.
	for _, sub := range []string{"payments", filepath.Join("payments", "ledger")} {
		initPath := filepath.Join(dir, "acme_apis", sub, "__init__.py")
		_, err := os.Stat(initPath)
		assert.NoError(t, err, "expected %s to exist", initPath)
	}

	// Leaf __init__.py exists.
	leafInit, err := os.ReadFile(filepath.Join(dir, "acme_apis", "payments", "ledger", "v1", "__init__.py"))
	require.NoError(t, err)
	assert.Contains(t, string(leafInit), "leaf package")
}

func TestScaffoldPythonPackage_NoDomain(t *testing.T) {
	dir := t.TempDir()

	err := ScaffoldPythonPackage(dir, "acme-orders-v1", "acme_apis.orders.v1")
	require.NoError(t, err)

	// Shorter hierarchy: acme_apis/orders/v1/
	leafInit, err := os.ReadFile(filepath.Join(dir, "acme_apis", "orders", "v1", "__init__.py"))
	require.NoError(t, err)
	assert.Contains(t, string(leafInit), "leaf package")

	pyproject, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(pyproject), `name = "acme-orders-v1"`)
}

func TestScaffoldPythonPackage_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Run twice — should not error.
	for i := 0; i < 2; i++ {
		err := ScaffoldPythonPackage(dir, "acme-payments-ledger-v1", "acme_apis.payments.ledger.v1")
		require.NoError(t, err, "iteration %d", i)
	}
}

func TestScaffoldPythonPackage_PyprojectFormat(t *testing.T) {
	dir := t.TempDir()

	err := ScaffoldPythonPackage(dir, "myorg-events-click-v0", "myorg_apis.events.click.v0")
	require.NoError(t, err)

	pyproject, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	require.NoError(t, err)

	content := string(pyproject)
	// Verify key sections exist.
	assert.True(t, strings.Contains(content, "[build-system]"))
	assert.True(t, strings.Contains(content, "[project]"))
	assert.True(t, strings.Contains(content, `name = "myorg-events-click-v0"`))
	assert.True(t, strings.Contains(content, `version = "0.0.0.dev0"`))
	assert.True(t, strings.Contains(content, `"grpcio>=1.60"`))
	assert.True(t, strings.Contains(content, `"protobuf>=4.25"`))
	assert.True(t, strings.Contains(content, `include = ["myorg_apis*"]`))
}
