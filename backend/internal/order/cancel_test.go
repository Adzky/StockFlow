package order_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
	"stockflow/backend/internal/order"
	"stockflow/backend/internal/testdb"
)

func TestCancel_RestoresStock(t *testing.T) {
	srv, pool := apitest.Start(t)
	testdb.InsertProduct(t, pool, "A", 1000, 10)
	testdb.InsertProduct(t, pool, "B", 1000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/orders",
		`{"customer":"Budi","items":[{"sku":"A","qty":3},{"sku":"B","qty":2}]}`,
	)
	require.Equal(t, http.StatusCreated, status)

	var created order.Order
	require.NoError(t, json.Unmarshal(body, &created))

	status, body = apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/orders/"+strconv.Itoa(created.ID)+"/cancel",
		"",
	)
	require.Equal(t, http.StatusOK, status)

	var cancelled order.Order
	require.NoError(t, json.Unmarshal(body, &cancelled))
	require.Equal(t, order.StatusCancelled, cancelled.Status)
	require.NotNil(t, cancelled.CancelledAt)

	require.Equal(t, 10, testdb.Int(
		t,
		pool,
		`SELECT stock FROM products WHERE sku = 'A'`,
	))
	require.Equal(t, 5, testdb.Int(
		t,
		pool,
		`SELECT stock FROM products WHERE sku = 'B'`,
	))

	require.Equal(t, 2, testdb.Int(
		t,
		pool,
		`SELECT count(*) FROM stock_movements
         WHERE type = 'in' AND order_id = $1`,
		created.ID,
	))
	require.Equal(t, 2, testdb.Int(
		t,
		pool,
		`SELECT count(*) FROM stock_movements
         WHERE type = 'in' AND order_id = $1
         AND note = $2`,
		created.ID,
		fmt.Sprintf("cancel order #%d", created.ID),
	))
}

func TestCancel_Twice409(t *testing.T) {
	srv, pool := apitest.Start(t)
	testdb.InsertProduct(t, pool, "A", 1000, 10)
	testdb.InsertProduct(t, pool, "B", 1000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/orders",
		`{"customer":"Budi","items":[{"sku":"A","qty":3},{"sku":"B","qty":2}]}`,
	)
	require.Equal(t, http.StatusCreated, status)

	var created order.Order
	require.NoError(t, json.Unmarshal(body, &created))

	path := "/api/orders/" + strconv.Itoa(created.ID) + "/cancel"

	status, _ = apitest.Do(t, srv, http.MethodPost, path, "")
	require.Equal(t, http.StatusOK, status)

	status, body = apitest.Do(t, srv, http.MethodPost, path, "")
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, order.CodeAlreadyCancelled, apitest.Err(t, body).Code)

	require.Equal(t, 10, testdb.Int(
		t,
		pool,
		`SELECT stock FROM products WHERE sku = 'A'`,
	))
	require.Equal(t, 5, testdb.Int(
		t,
		pool,
		`SELECT stock FROM products WHERE sku = 'B'`,
	))
}

func TestCancel_404(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/orders/999/cancel",
		"",
	)

	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, "NOT_FOUND", apitest.Err(t, body).Code)
}

func TestCancel_ConcurrentOnlyOnce(t *testing.T) {
	srv, pool := apitest.Start(t)
	testdb.InsertProduct(t, pool, "A", 1000, 10)
	testdb.InsertProduct(t, pool, "B", 1000, 5)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodPost,
		"/api/orders",
		`{"customer":"Budi","items":[{"sku":"A","qty":3},{"sku":"B","qty":2}]}`,
	)
	require.Equal(t, http.StatusCreated, status)

	var created order.Order
	require.NoError(t, json.Unmarshal(body, &created))

	const n = 10
	var ok, conflict atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	path := "/api/orders/" + strconv.Itoa(created.ID) + "/cancel"

	for range n {
		wg.Go(func() {
			<-start

			status, _ := apitest.Do(t, srv, http.MethodPost, path, "")
			switch status {
			case http.StatusOK:
				ok.Add(1)
			case http.StatusConflict:
				conflict.Add(1)
			default:
				t.Errorf("unexpected status %d", status)
			}
		})
	}

	close(start)
	wg.Wait()

	require.EqualValues(t, 1, ok.Load())
	require.EqualValues(t, 9, conflict.Load())
	require.Equal(t, 10, testdb.Int(
		t,
		pool,
		`SELECT stock FROM products WHERE sku = 'A'`,
	))
	require.Equal(t, 5, testdb.Int(
		t,
		pool,
		`SELECT stock FROM products WHERE sku = 'B'`,
	))
	require.Equal(t, 2, testdb.Int(
		t,
		pool,
		`SELECT count(*) FROM stock_movements
         WHERE type = 'in' AND order_id = $1`,
		created.ID,
	))
}
