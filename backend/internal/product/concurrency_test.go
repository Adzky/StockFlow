package product_test

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
	"stockflow/backend/internal/testdb"
)

func TestAdjustStock_ConcurrentOutNeverNegative(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "CONC-1", 1000, 10)

	const n = 50
	var ok, conflict atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for range n {
		wg.Go(func() {
			<-start

			status, _ := apitest.Do(
				t,
				srv,
				http.MethodPost,
				fmt.Sprintf("/api/products/%d/stock", id),
				`{"type":"out","qty":1}`,
			)

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

	require.EqualValues(t, 10, ok.Load())
	require.EqualValues(t, 40, conflict.Load())
	require.Equal(
		t,
		0,
		testdb.Int(t, pool, `SELECT stock FROM products WHERE id = $1`, id),
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

func TestAdjustStock_ConcurrentMixedInOut(t *testing.T) {
	srv, pool := apitest.Start(t)
	id := testdb.InsertProduct(t, pool, "MIX-1", 1000, 0)

	const n = 20
	var ins, outs atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	request := func(body string, counter *atomic.Int32) {
		defer wg.Done()
		<-start

		status, _ := apitest.Do(
			t,
			srv,
			http.MethodPost,
			fmt.Sprintf("/api/products/%d/stock", id),
			body,
		)

		switch status {
		case http.StatusOK:
			counter.Add(1)
		case http.StatusConflict:
			// `out` boleh konflik karena stok berasal dari request `in`.
		default:
			t.Errorf("unexpected status %d", status)
		}
	}

	for range n {
		wg.Add(1)
		go request(`{"type":"in","qty":1}`, &ins)
	}

	for range n {
		wg.Add(1)
		go request(`{"type":"out","qty":1}`, &outs)
	}

	close(start)
	wg.Wait()

	successfulIn := ins.Load()
	successfulOut := outs.Load()
	movementCount := testdb.Int(
		t,
		pool,
		`SELECT count(*) FROM stock_movements WHERE product_id = $1`,
		id,
	)
	finalStock := testdb.Int(
		t,
		pool,
		`SELECT stock FROM products WHERE id = $1`,
		id,
	)

	require.Equal(
		t,
		successfulIn-successfulOut,
		int32(finalStock),
	)
	require.GreaterOrEqual(t, finalStock, 0)
	require.Equal(
		t,
		int(successfulIn+successfulOut),
		movementCount,
	)
}
