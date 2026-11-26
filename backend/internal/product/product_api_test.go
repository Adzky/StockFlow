package product_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
	"stockflow/backend/internal/httpx"
	"stockflow/backend/internal/product"
	"stockflow/backend/internal/testdb"
)

func TestCreateProduct_201(t *testing.T) {
	srv, pool := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products",
		`{"sku":"brs-1","name":"Beras","price":78000,"stock":5}`,
	)

	require.Equal(t, http.StatusCreated, status)

	var created product.Product
	require.NoError(t, json.Unmarshal(body, &created))
	require.Equal(t, "BRS-1", created.SKU)
	require.Equal(t, 5, created.Stock)
	require.True(t, created.IsActive)
	require.NotNil(t, created.LastMovementAt)

	movementCount := testdb.Int(
		t,
		pool,
		`SELECT count(*) FROM stock_movements
		 WHERE type = 'in' AND note = 'initial stock'`,
	)
	require.Equal(t, 1, movementCount)
}

func TestCreateProduct_NoStockNoMovement(t *testing.T) {
	srv, pool := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products",
		`{"sku":"brs-1","name":"Beras","price":78000}`,
	)

	require.Equal(t, http.StatusCreated, status)

	var created product.Product
	require.NoError(t, json.Unmarshal(body, &created))
	require.Equal(t, 0, created.Stock)
	require.Nil(t, created.LastMovementAt)

	movementCount := testdb.Int(
		t,
		pool,
		`SELECT count(*) FROM stock_movements WHERE product_id = $1`,
		created.ID,
	)
	require.Zero(t, movementCount)
}

func TestCreateProduct_DuplicateSKU409(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, _ := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products",
		`{"sku":"BRS-1","name":"Beras","price":78000}`,
	)
	require.Equal(t, http.StatusCreated, status)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products",
		`{"sku":" brs-1 ","name":"Beras lain","price":80000}`,
	)

	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, "SKU_ALREADY_EXISTS", apitest.Err(t, body).Code)
}

func TestCreateProduct_Validation400(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products",
		`{"sku":"","name":"","price":-1}`,
	)

	require.Equal(t, http.StatusBadRequest, status)
	err := apitest.Err(t, body)
	require.Equal(t, "VALIDATION_FAILED", err.Code)

	fields := make([]string, 0, len(err.Details))
	for _, detail := range err.Details {
		fields = append(fields, detail.Field)
	}
	require.Equal(t, []string{"sku", "name", "price"}, fields)
}

func TestCreateProduct_StringPrice400(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/products",
		`{"sku":"BRS-1","name":"Beras","price":"10"}`,
	)

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "INVALID_JSON", apitest.Err(t, body).Code)
}

func TestUpdateProduct_200(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPut,
		"/api/products/"+strconv.Itoa(id),
		`{"name":"Beras Super","price":80000,"is_active":false}`,
	)

	require.Equal(t, http.StatusOK, status)

	var updated product.Product
	require.NoError(t, json.Unmarshal(body, &updated))
	require.Equal(t, id, updated.ID)
	require.Equal(t, "BRS-1", updated.SKU)
	require.Equal(t, "Beras Super", updated.Name)
	require.Equal(t, 80000, updated.Price)
	require.Equal(t, 5, updated.Stock)
	require.False(t, updated.IsActive)
}

func TestUpdateProduct_404(t *testing.T) {
	srv, _ := apitest.Start(t)

	for _, path := range []string{
		"/api/products/999",
		"/api/products/abc",
	} {
		t.Run(path, func(t *testing.T) {
			status, body := apitest.Do(
				t,
				srv,
				http.MethodPut,
				path,
				`{"name":"Beras","price":78000,"is_active":true}`,
			)

			require.Equal(t, http.StatusNotFound, status)
			require.Equal(t, "NOT_FOUND", apitest.Err(t, body).Code)
		})
	}
}

func TestUpdateProduct_RejectsStockAndSKU(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 5)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "stock",
			body: `{"name":"Beras","price":80000,"is_active":true,"stock":9}`,
		},
		{
			name: "sku",
			body: `{"name":"Beras","price":80000,"is_active":true,"sku":"OTHER"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := apitest.Do(
				t,
				srv,
				http.MethodPut,
				"/api/products/"+strconv.Itoa(id),
				tt.body,
			)

			require.Equal(t, http.StatusBadRequest, status)
			require.Equal(t, "INVALID_JSON", apitest.Err(t, body).Code)
		})
	}

	require.Equal(
		t,
		5,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE id = $1`, id),
	)

	var sku string
	require.NoError(t, pool.QueryRow(
		t.Context(),
		`SELECT sku FROM products WHERE id = $1`,
		id,
	).Scan(&sku))
	require.Equal(t, "BRS-1", sku)
}

func listFixture(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()

	srv, pool := apitest.Start(t)

	insert := func(sku, name string, stock int) int {
		id := testdb.InsertProduct(t, pool, sku, 100, stock)
		testdb.Exec(t, pool, `UPDATE products SET name = $2 WHERE id = $1`, id, name)
		return id
	}

	insert("BRS-1", "Beras Premium", 50)
	insert("MYK-1", "Minyak Goreng", 5)

	gulID := insert("GUL-1", "Gula Pasir", 10)
	testdb.Exec(t, pool, `UPDATE products SET is_active = false WHERE id = $1`, gulID)

	insert("PCT-50", "Diskon 50% Pack", 20)

	for i := 1; i <= 21; i++ {
		insert(fmt.Sprintf("BULK-%02d", i), "Bulk item", 100)
	}

	return srv, pool
}

func decodeProductPage(
	t *testing.T,
	raw []byte,
) httpx.Paged[product.Product] {
	t.Helper()

	var page httpx.Paged[product.Product]
	require.NoError(t, json.Unmarshal(raw, &page))
	return page
}

func TestListProducts_DefaultPaging(t *testing.T) {
	srv, _ := listFixture(t)

	status, raw := apitest.Do(t, srv, http.MethodGet, "/api/products", "")
	require.Equal(t, http.StatusOK, status)

	page := decodeProductPage(t, raw)
	require.Equal(t, 1, page.Page)
	require.Equal(t, 20, page.Limit)
	require.Equal(t, 25, page.Total)
	require.Len(t, page.Data, 20)

	for i := 1; i < len(page.Data); i++ {
		require.Less(t, page.Data[i-1].ID, page.Data[i].ID)
	}
}

func TestDeleteProduct_NoHistory204(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 0)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodDelete,
		"/api/products/"+strconv.Itoa(id),
		"",
	)

	require.Equal(t, http.StatusNoContent, status)
	require.Empty(t, body)
	require.Zero(
		t,
		testdb.Int(t, pool, `SELECT count(*) FROM products WHERE id = $1`, id),
	)
}

func TestDeleteProduct_WithHistory409(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "BRS-1", 78000, 0)

	testdb.Exec(
		t,
		pool,
		`INSERT INTO stock_movements (product_id, type, qty, note)
         VALUES ($1, 'in', 1, 'test movement')`,
		id,
	)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodDelete,
		"/api/products/"+strconv.Itoa(id),
		"",
	)

	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, "PRODUCT_HAS_HISTORY", apitest.Err(t, body).Code)
	require.Equal(
		t,
		1,
		testdb.Int(t, pool, `SELECT count(*) FROM products WHERE id = $1`, id),
	)
	require.Equal(
		t,
		1,
		testdb.Int(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1`, id),
	)
}

func TestDeleteProduct_404(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodDelete,
		"/api/products/999",
		"",
	)

	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, "NOT_FOUND", apitest.Err(t, body).Code)
}
