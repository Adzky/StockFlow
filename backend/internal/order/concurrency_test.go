package order_test

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
	"stockflow/backend/internal/testdb"
)

func TestCreateOrder_ConcurrentNeverOversells(t *testing.T) {
	srv, pool := apitest.Start(t)
	testdb.InsertProduct(t, pool, "CONC-1", 1000, 10)

	const n = 50
	var created, conflict atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for range n {
		wg.Go(func() {
			<-start

			status, _ := apitest.Do(
				t,
				srv,
				http.MethodPost,
				"/api/orders",
				`{"customer":"Concurrent","items":[{"sku":"CONC-1","qty":1}]}`,
			)

			switch status {
			case http.StatusCreated:
				created.Add(1)
			case http.StatusConflict:
				conflict.Add(1)
			default:
				t.Errorf("unexpected status %d", status)
			}
		})
	}

	close(start)
	wg.Wait()

	require.EqualValues(t, 10, created.Load())
	require.EqualValues(t, 40, conflict.Load())
	require.Equal(
		t,
		0,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE sku = 'CONC-1'`),
	)
	require.Equal(
		t,
		10,
		testdb.Int(t, pool, `SELECT count(*) FROM sales_orders`),
	)
	require.Equal(
		t,
		10,
		testdb.Int(
			t,
			pool,
			`SELECT count(*) FROM stock_movements WHERE type = 'out'`,
		),
	)
}

func TestCreateOrder_ConcurrentMultiItemAtomic(t *testing.T) {
	srv, pool := apitest.Start(t)
	testdb.InsertProduct(t, pool, "A", 1000, 100)
	testdb.InsertProduct(t, pool, "B", 1000, 5)

	const n = 20
	var created, conflict atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for range n {
		wg.Go(func() {
			<-start

			status, _ := apitest.Do(
				t,
				srv,
				http.MethodPost,
				"/api/orders",
				`{"customer":"Atomic","items":[{"sku":"A","qty":1},{"sku":"B","qty":1}]}`,
			)

			switch status {
			case http.StatusCreated:
				created.Add(1)
			case http.StatusConflict:
				conflict.Add(1)
			default:
				t.Errorf("unexpected status %d", status)
			}
		})
	}

	close(start)
	wg.Wait()

	require.EqualValues(t, 5, created.Load())
	require.EqualValues(t, 15, conflict.Load())
	require.Equal(
		t,
		95,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE sku = 'A'`),
	)
	require.Equal(
		t,
		0,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE sku = 'B'`),
	)
}

func TestCreateOrder_OppositeOrderNoDeadlock(t *testing.T) {
	srv, pool := apitest.Start(t)
	testdb.InsertProduct(t, pool, "A", 1000, 1000)
	testdb.InsertProduct(t, pool, "B", 1000, 1000)

	const n = 40
	var created atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := range n {
		i := i

		wg.Go(func() {
			<-start

			body := `{"customer":"No deadlock","items":[{"sku":"A","qty":1},{"sku":"B","qty":1}]}`
			if i%2 == 1 {
				body = `{"customer":"No deadlock","items":[{"sku":"B","qty":1},{"sku":"A","qty":1}]}`
			}

			status, _ := apitest.Do(
				t,
				srv,
				http.MethodPost,
				"/api/orders",
				body,
			)

			if status == http.StatusCreated {
				created.Add(1)
			} else {
				t.Errorf("unexpected status %d", status)
			}
		})
	}

	close(start)
	wg.Wait()

	require.EqualValues(t, 40, created.Load())
	require.Equal(
		t,
		960,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE sku = 'A'`),
	)
	require.Equal(
		t,
		960,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE sku = 'B'`),
	)
}
