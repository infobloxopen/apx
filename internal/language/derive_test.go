package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveAllCoords_WithOrg(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	ctx := DerivationContext{
		SourceRepo: "github.com/acme/apis",
		Org:        "acme",
		API:        api,
	}

	coords, err := DeriveAllCoords(ctx)
	require.NoError(t, err)

	// Should have all 6 languages
	assert.Len(t, coords, 6)

	// Go
	goCoords, ok := coords["go"]
	require.True(t, ok)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", goCoords.Module)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v1", goCoords.Import)

	// C++
	cppCoords, ok := coords["cpp"]
	require.True(t, ok)
	assert.Equal(t, "acme-payments-ledger-v1-proto", cppCoords.Module)
	assert.Equal(t, "acme::payments::ledger::v1", cppCoords.Import)

	// Java
	javaCoords, ok := coords["java"]
	require.True(t, ok)
	assert.Equal(t, "com.acme.apis:payments-ledger-v1-proto", javaCoords.Module)
	assert.Equal(t, "com.acme.apis.payments.ledger.v1", javaCoords.Import)

	// Python
	pyCoords, ok := coords["python"]
	require.True(t, ok)
	assert.Equal(t, "acme-payments-ledger-v1", pyCoords.Module)
	assert.Equal(t, "acme_apis.payments.ledger.v1", pyCoords.Import)

	// Rust
	rustCoords, ok := coords["rust"]
	require.True(t, ok)
	assert.Equal(t, "acme-payments-ledger-v1-proto", rustCoords.Module)
	assert.Equal(t, "acme_payments::ledger::v1", rustCoords.Import)

	// TypeScript
	tsCoords, ok := coords["typescript"]
	require.True(t, ok)
	assert.Equal(t, "@acme/payments-ledger-v1-proto", tsCoords.Module)
	assert.Equal(t, "@acme/payments-ledger-v1-proto", tsCoords.Import)
}

func TestDeriveAllCoords_WithoutOrg(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	ctx := DerivationContext{
		SourceRepo: "github.com/acme/apis",
		Org:        "", // no org
		API:        api,
	}

	coords, err := DeriveAllCoords(ctx)
	require.NoError(t, err)

	// Only Go should be present
	assert.Len(t, coords, 1)
	_, ok := coords["go"]
	assert.True(t, ok)
	_, ok = coords["python"]
	assert.False(t, ok)
}

func TestDeriveAllCoords_WithImportRoot(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	ctx := DerivationContext{
		SourceRepo: "github.com/acme/apis",
		ImportRoot: "go.acme.dev/apis",
		Org:        "acme",
		API:        api,
	}

	coords, err := DeriveAllCoords(ctx)
	require.NoError(t, err)

	// Go should use the import root
	goCoords := coords["go"]
	assert.Equal(t, "go.acme.dev/apis/proto/payments/ledger", goCoords.Module)
	assert.Equal(t, "go.acme.dev/apis/proto/payments/ledger/v1", goCoords.Import)

	// Python should not be affected by import root
	pyCoords := coords["python"]
	assert.Equal(t, "acme-payments-ledger-v1", pyCoords.Module)
}

func TestDeriveAllCoords_ThreePart(t *testing.T) {
	api, err := config.ParseAPIID("proto/orders/v1")
	require.NoError(t, err)

	ctx := DerivationContext{
		SourceRepo: "github.com/acme/apis",
		Org:        "acme",
		API:        api,
	}

	coords, err := DeriveAllCoords(ctx)
	require.NoError(t, err)

	// Go — no domain segment
	assert.Equal(t, "github.com/acme/apis/proto/orders", coords["go"].Module)

	// C++ — no domain
	assert.Equal(t, "acme-orders-v1-proto", coords["cpp"].Module)
	assert.Equal(t, "acme::orders::v1", coords["cpp"].Import)

	// Java — no domain
	assert.Equal(t, "com.acme.apis:orders-v1-proto", coords["java"].Module)

	// Python — no domain
	assert.Equal(t, "acme-orders-v1", coords["python"].Module)

	// Rust — no domain
	assert.Equal(t, "acme-orders-v1-proto", coords["rust"].Module)
	assert.Equal(t, "acme_orders::v1", coords["rust"].Import)

	// TypeScript — no domain
	assert.Equal(t, "@acme/orders-v1-proto", coords["typescript"].Module)
}

func TestDeriveAllCoords_V2Line(t *testing.T) {
	api, err := config.ParseAPIID("proto/inventory/products/v2")
	require.NoError(t, err)

	ctx := DerivationContext{
		SourceRepo: "github.com/acme/apis",
		Org:        "acme",
		API:        api,
	}

	coords, err := DeriveAllCoords(ctx)
	require.NoError(t, err)

	// Go v2+ includes major version suffix in module path
	assert.Equal(t, "github.com/acme/apis/proto/inventory/products/v2", coords["go"].Module)
	assert.Equal(t, "github.com/acme/apis/proto/inventory/products/v2", coords["go"].Import)
}
