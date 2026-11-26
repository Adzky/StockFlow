package product_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
	"stockflow/backend/internal/product"
	"stockflow/backend/internal/testdb"
)

type adjustResult struct {
	Product  product.Product  `json:"product"`
	Movement product.Movement `json:"movement"`
}

func TestAdjustStock_OutInsufficient_NothingWritten(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products/"+strconv.Itoa(id)+"/stock",
		`{"type":"out","qty":6}`,
	)

	require.Equal(t, http.StatusConflict, status)
	err := apitest.Err(t, body)
	require.Equal(t, "INSUFFICIENT_STOCK", err.Code)
	require.NotNil(t, err.Available)
	require.Equal(t, 5, *err.Available)
	require.Empty(t, err.SKU)

	require.Equal(
		t,
		5,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE id = $1`, id),
	)
	require.Zero(
		t,
		testdb.Int(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1`, id),
	)
}

func TestAdjustStock_InHappyPath(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products/"+strconv.Itoa(id)+"/stock",
		`{"type":"in","qty":3,"note":"restock"}`,
	)

	require.Equal(t, http.StatusOK, status)

	var result adjustResult
	require.NoError(t, json.Unmarshal(body, &result))
	require.Equal(t, 8, result.Product.Stock)
	require.Equal(t, product.MovementIn, result.Movement.Type)
	require.Equal(t, 3, result.Movement.Qty)
	require.NotNil(t, result.Movement.Note)
	require.Equal(t, "restock", *result.Movement.Note)

	require.Equal(
		t,
		1,
		testdb.Int(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1`, id),
	)
	require.NotNil(t, result.Product.LastMovementAt)
	require.Equal(t, result.Movement.CreatedAt, *result.Product.LastMovementAt)
}

func TestAdjustStock_OutHappyPath(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products/"+strconv.Itoa(id)+"/stock",
		`{"type":"out","qty":2,"note":"sale"}`,
	)

	require.Equal(t, http.StatusOK, status)

	var result adjustResult
	require.NoError(t, json.Unmarshal(body, &result))
	require.Equal(t, 3, result.Product.Stock)
	require.Equal(t, product.MovementOut, result.Movement.Type)
	require.Equal(t, 2, result.Movement.Qty)
	require.Equal(
		t,
		1,
		testdb.Int(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1`, id),
	)
}

func TestAdjustStock_OutExactlyAll(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products/"+strconv.Itoa(id)+"/stock",
		`{"type":"out","qty":5}`,
	)

	require.Equal(t, http.StatusOK, status)

	var result adjustResult
	require.NoError(t, json.Unmarshal(body, &result))
	require.Zero(t, result.Product.Stock)
	require.Equal(
		t,
		1,
		testdb.Int(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1`, id),
	)
}

func TestAdjustStock_404(t *testing.T) {
	srv, _ := apitest.Start(t)

	for _, body := range []string{
		`{"type":"in","qty":1}`,
		`{"type":"out","qty":1}`,
	} {
		status, raw := apitest.Do(
			t,
			srv,
			http.MethodPost,
			"/api/products/999/stock",
			body,
		)

		require.Equal(t, http.StatusNotFound, status)
		require.Equal(t, "NOT_FOUND", apitest.Err(t, raw).Code)
	}
}

func TestAdjustStock_Validation400(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	tests := []struct {
		name     string
		body     string
		wantCode string
	}{
		{name: "zero quantity", body: `{"type":"in","qty":0}`, wantCode: "VALIDATION_FAILED"},
		{name: "negative quantity", body: `{"type":"in","qty":-1}`, wantCode: "VALIDATION_FAILED"},
		{name: "invalid type", body: `{"type":"x","qty":1}`, wantCode: "VALIDATION_FAILED"},
		{name: "quantity above range", body: `{"type":"in","qty":1000001}`, wantCode: "VALIDATION_FAILED"},
		{name: "fractional quantity", body: `{"type":"in","qty":1.5}`, wantCode: "INVALID_JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, raw := apitest.Do(
				t,
				srv,
				http.MethodPost,
				"/api/products/"+strconv.Itoa(id)+"/stock",
				tt.body,
			)

			require.Equal(t, http.StatusBadRequest, status)
			require.Equal(t, tt.wantCode, apitest.Err(t, raw).Code)
		})
	}

	require.Equal(
		t,
		5,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE id = $1`, id),
	)
	require.Zero(
		t,
		testdb.Int(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1`, id),
	)
}

func TestAdjustStock_InactiveProductAllowed(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)
	testdb.Exec(t, pool, `UPDATE products SET is_active = false WHERE id = $1`, id)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products/"+strconv.Itoa(id)+"/stock",
		`{"type":"in","qty":1}`,
	)

	require.Equal(t, http.StatusOK, status)

	var result adjustResult
	require.NoError(t, json.Unmarshal(body, &result))
	require.False(t, result.Product.IsActive)
	require.Equal(t, 6, result.Product.Stock)
}

func TestMovements_NewestFirst(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	var id1, id2, id3 int

	err := pool.QueryRow(
		t.Context(),
		`INSERT INTO stock_movements (product_id, type, qty, note, created_at)
         VALUES ($1, 'in', 1, 'first', '2026-01-01T00:00:00Z')
         RETURNING id`,
		id,
	).Scan(&id1)
	require.NoError(t, err)

	err = pool.QueryRow(
		t.Context(),
		`INSERT INTO stock_movements (product_id, type, qty, note, created_at)
         VALUES ($1, 'out', 1, 'second', '2026-01-03T00:00:00Z')
         RETURNING id`,
		id,
	).Scan(&id2)
	require.NoError(t, err)

	err = pool.QueryRow(
		t.Context(),
		`INSERT INTO stock_movements (product_id, type, qty, note, created_at)
         VALUES ($1, 'in', 2, 'third', '2026-01-03T00:00:00Z')
         RETURNING id`,
		id,
	).Scan(&id3)
	require.NoError(t, err)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/products/"+strconv.Itoa(id)+"/movements",
		"",
	)

	require.Equal(t, http.StatusOK, status)

	var result struct {
		Data []product.Movement `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &result))

	require.Len(t, result.Data, 3)
	require.Equal(t, id3, result.Data[0].ID)
	require.Equal(t, id2, result.Data[1].ID)
	require.Equal(t, id1, result.Data[2].ID)
}

func TestMovements_Empty(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/products/"+strconv.Itoa(id)+"/movements",
		"",
	)

	require.Equal(t, http.StatusOK, status)

	var result struct {
		Data []product.Movement `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &result))
	require.NotNil(t, result.Data)
	require.Empty(t, result.Data)
}

func TestMovements_404(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/products/999/movements",
		"",
	)

	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, "NOT_FOUND", apitest.Err(t, body).Code)
}
