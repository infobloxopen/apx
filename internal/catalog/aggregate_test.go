package catalog

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateSource_Empty(t *testing.T) {
	agg := &AggregateSource{}
	cat, err := agg.Load()
	require.NoError(t, err)
	assert.Equal(t, 1, cat.Version)
	assert.Empty(t, cat.Modules)
}

func TestAggregateSource_SingleSource(t *testing.T) {
	agg := &AggregateSource{
		Sources: []CatalogSource{
			&stubSource{
				name: "src1",
				cat: &Catalog{Org: "acme", Repo: "apis", Modules: []Module{
					{ID: "proto/payments/ledger/v1", Format: "proto"},
				}},
			},
		},
	}

	cat, err := agg.Load()
	require.NoError(t, err)
	require.Len(t, cat.Modules, 1)
	assert.Equal(t, "proto/payments/ledger/v1", cat.Modules[0].ID)
}

func TestAggregateSource_MergeMultiple(t *testing.T) {
	agg := &AggregateSource{
		Sources: []CatalogSource{
			&stubSource{
				name: "src1",
				cat: &Catalog{Org: "acme", Repo: "apis", Modules: []Module{
					{ID: "proto/payments/ledger/v1", Format: "proto"},
				}},
			},
			&stubSource{
				name: "src2",
				cat: &Catalog{Org: "acme", Repo: "shared", Modules: []Module{
					{ID: "proto/billing/invoices/v1", Format: "proto"},
				}},
			},
		},
	}

	cat, err := agg.Load()
	require.NoError(t, err)
	require.Len(t, cat.Modules, 2)
	assert.Equal(t, "proto/payments/ledger/v1", cat.Modules[0].ID)
	assert.Equal(t, "proto/billing/invoices/v1", cat.Modules[1].ID)
}

func TestAggregateSource_Dedup_LeftmostWins(t *testing.T) {
	agg := &AggregateSource{
		Sources: []CatalogSource{
			&stubSource{
				name: "src1",
				cat: &Catalog{Org: "acme", Repo: "apis", Modules: []Module{
					{ID: "proto/payments/ledger/v1", Format: "proto", Version: "v1.0.0"},
				}},
			},
			&stubSource{
				name: "src2",
				cat: &Catalog{Org: "acme", Repo: "apis", Modules: []Module{
					{ID: "proto/payments/ledger/v1", Format: "proto", Version: "v2.0.0"},
				}},
			},
		},
	}

	cat, err := agg.Load()
	require.NoError(t, err)
	require.Len(t, cat.Modules, 1, "duplicate module should be deduplicated")
	assert.Equal(t, "v1.0.0", cat.Modules[0].Version, "leftmost source should win")
}

func TestAggregateSource_PartialFailure(t *testing.T) {
	agg := &AggregateSource{
		Sources: []CatalogSource{
			&stubSource{name: "failing", err: fmt.Errorf("network error")},
			&stubSource{
				name: "working",
				cat: &Catalog{Org: "acme", Repo: "apis", Modules: []Module{
					{ID: "proto/orders/v1", Format: "proto"},
				}},
			},
		},
	}

	cat, err := agg.Load()
	require.NoError(t, err, "partial failure should not fail the aggregate")
	require.Len(t, cat.Modules, 1)
}

func TestAggregateSource_AllFail(t *testing.T) {
	agg := &AggregateSource{
		Sources: []CatalogSource{
			&stubSource{name: "src1", err: fmt.Errorf("error 1")},
			&stubSource{name: "src2", err: fmt.Errorf("error 2")},
		},
	}

	_, err := agg.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all catalog sources failed")
	assert.Contains(t, err.Error(), "error 1")
	assert.Contains(t, err.Error(), "error 2")
}

func TestAggregateSource_Name(t *testing.T) {
	agg := &AggregateSource{
		Sources: []CatalogSource{
			&stubSource{name: "src1"},
			&stubSource{name: "src2"},
		},
	}
	assert.Equal(t, "aggregate[src1, src2]", agg.Name())
}
