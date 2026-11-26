package order_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
	"stockflow/backend/internal/apperr"
	"stockflow/backend/internal/order"
	"stockflow/backend/internal/testdb"
)

func TestPostOrder_201MatchesGet(t *testing.T) {
	srv, pool := apitest.Start(t)

	testdb.InsertProduct(t, pool, "A", 1000, 10)
	testdb.InsertProduct(t, pool, "B", 2500, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/orders",
		`{"customer":"Budi","items":[{"sku":"A","qty":3},{"sku":"B","qty":2}]}`,
	)
	require.Equal(t, http.StatusCreated, status)

	var posted order.Order
	require.NoError(t, json.Unmarshal(body, &posted))
	require.NotZero(t, posted.ID)
	require.Equal(t, int64(8000), posted.Total)

	status, body = apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/orders/"+strconv.Itoa(posted.ID),
		"",
	)
	require.Equal(t, http.StatusOK, status)

	var fetched order.Order
	require.NoError(t, json.Unmarshal(body, &fetched))
	require.Equal(t, posted, fetched)
}

func TestPostOrder_ClientPricesRejected(t *testing.T) {
	srv, pool := apitest.Start(t)
	testdb.InsertProduct(t, pool, "A", 1000, 10)

	tests := []string{
		`{"customer":"Budi","items":[{"sku":"A","qty":1,"unit_price":1}]}`,
		`{"customer":"Budi","total":1,"items":[{"sku":"A","qty":1}]}`,
	}

	for _, body := range tests {
		status, raw := apitest.Do(
			t,
			srv,
			http.MethodPost,
			"/api/orders",
			body,
		)

		require.Equal(t, http.StatusBadRequest, status)
		require.Equal(t, "INVALID_JSON", apitest.Err(t, raw).Code)
	}

	require.Zero(t, testdb.Int(t, pool, `SELECT count(*) FROM sales_orders`))
	require.Zero(t, testdb.Int(t, pool, `SELECT count(*) FROM sales_order_items`))
}

func TestPostOrder_409EnvelopeShape(t *testing.T) {
	srv, pool := apitest.Start(t)

	testdb.InsertProduct(t, pool, "A", 1000, 10)
	testdb.InsertProduct(t, pool, "B", 100, 1)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/orders",
		`{"customer":"Budi","items":[{"sku":"A","qty":1},{"sku":"B","qty":5}]}`,
	)

	require.Equal(t, http.StatusConflict, status)

	var envelope map[string]map[string]any
	require.NoError(t, json.Unmarshal(body, &envelope))

	require.Equal(t, "INSUFFICIENT_STOCK", envelope["error"]["code"])
	require.Equal(t, "B", envelope["error"]["sku"])
	require.Equal(t, float64(1), envelope["error"]["available"])
}

func TestPostOrder_400Messages(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/orders",
		`{"customer":"","items":[]}`,
	)

	require.Equal(t, http.StatusBadRequest, status)

	err := apitest.Err(t, body)
	require.Equal(t, "VALIDATION_FAILED", err.Code)
	require.Equal(t, []apperr.Detail{
		{Field: "customer", Message: "is required"},
		{Field: "items", Message: "must contain at least one item"},
	}, err.Details)
}

func TestGetOrder_404(t *testing.T) {
	srv, _ := apitest.Start(t)

	for _, path := range []string{
		"/api/orders/999",
		"/api/orders/abc",
	} {
		status, body := apitest.Do(
			t,
			srv,
			http.MethodGet,
			path,
			"",
		)

		require.Equal(t, http.StatusNotFound, status)
		require.Equal(t, "NOT_FOUND", apitest.Err(t, body).Code)
	}
}
