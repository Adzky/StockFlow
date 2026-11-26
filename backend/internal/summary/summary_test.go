package summary_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
	"stockflow/backend/internal/testdb"
)

type testSummary struct {
	TotalProducts   int   `json:"total_products"`
	ActiveProducts  int   `json:"active_products"`
	TotalStockValue int64 `json:"total_stock_value"`
	LowStockCount   int   `json:"low_stock_count"`
}

func TestSummary_Empty(t *testing.T) {
	srv, _ := apitest.Start(t)
	status, body := apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/summary",
		"",
	)

	require.Equal(t, http.StatusOK, status)

	var result testSummary
	require.NoError(t, json.Unmarshal(body, &result))
	require.Equal(t, testSummary{}, result)
}

func TestSummary_Values(t *testing.T) {
	srv, pool := apitest.Start(t)

	testdb.InsertProduct(t, pool, "A", 1000, 5)
	testdb.InsertProduct(t, pool, "B", 2000, 20)

	inactiveID := testdb.InsertProduct(t, pool, "C", 500, 3)
	testdb.Exec(
		t,
		pool,
		`UPDATE products SET is_active = false WHERE id = $1`,
		inactiveID,
	)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/summary",
		"",
	)
	require.Equal(t, http.StatusOK, status)

	var result testSummary
	require.NoError(t, json.Unmarshal(body, &result))
	require.Equal(t, testSummary{
		TotalProducts:   3,
		ActiveProducts:  2,
		TotalStockValue: 45000,
		LowStockCount:   1,
	}, result)
}

func TestSummary_NoInt32Overflow(t *testing.T) {
	srv, pool := apitest.Start(t)
	testdb.InsertProduct(t, pool, "BIG", 1_000_000_000, 1_000_000)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/summary",
		"",
	)
	require.Equal(t, http.StatusOK, status)

	var result testSummary
	require.NoError(t, json.Unmarshal(body, &result))
	require.Equal(t, int64(1_000_000_000_000_000), result.TotalStockValue)
}
