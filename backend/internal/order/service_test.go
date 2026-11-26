package order_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
	"stockflow/backend/internal/apperr"
	"stockflow/backend/internal/order"
	"stockflow/backend/internal/testdb"
)

func intPtr(v int) *int {
	return &v
}

func TestCreate_HappyPath(t *testing.T) {
	_, pool := apitest.Start(t)
	service := order.NewService(pool)

	idA := testdb.InsertProduct(t, pool, "A", 1000, 10)
	idB := testdb.InsertProduct(t, pool, "B", 2500, 5)
	require.NotZero(t, idA)
	require.NotZero(t, idB)

	created, err := service.Create(t.Context(), order.CreateInput{
		Customer: "Budi",
		Items: []order.ItemInput{
			{SKU: "A", Qty: intPtr(3)},
			{SKU: "B", Qty: intPtr(2)},
		},
	})
	require.NoError(t, err)

	require.Equal(t, int64(8000), created.Total)
	require.Len(t, created.Items, 2)
	require.Equal(t, order.Item{
		ID:        created.Items[0].ID,
		SKU:       "A",
		Name:      "A",
		Qty:       3,
		UnitPrice: 1000,
		LineTotal: 3000,
	}, created.Items[0])
	require.Equal(t, order.Item{
		ID:        created.Items[1].ID,
		SKU:       "B",
		Name:      "B",
		Qty:       2,
		UnitPrice: 2500,
		LineTotal: 5000,
	}, created.Items[1])

	require.Equal(t, 7, testdb.Int(
		t, pool, `SELECT stock FROM products WHERE sku = 'A'`,
	))
	require.Equal(t, 3, testdb.Int(
		t, pool, `SELECT stock FROM products WHERE sku = 'B'`,
	))

	require.Equal(t, 2, testdb.Int(
		t, pool,
		`SELECT count(*) FROM stock_movements
         WHERE type = 'out' AND order_id = $1`,
		created.ID,
	))
	require.Equal(t, 2, testdb.Int(
		t, pool,
		`SELECT count(*) FROM stock_movements
         WHERE type = 'out' AND order_id = $1
         AND note = $2`,
		created.ID,
		"order #"+fmt.Sprintf("%d", created.ID),
	))
}

func TestCreate_OneShortItem_NothingWritten(t *testing.T) {
	_, pool := apitest.Start(t)
	service := order.NewService(pool)

	testdb.InsertProduct(t, pool, "A", 100, 10)
	testdb.InsertProduct(t, pool, "B", 200, 1)
	testdb.InsertProduct(t, pool, "C", 300, 10)

	_, err := service.Create(t.Context(), order.CreateInput{
		Customer: "Budi",
		Items: []order.ItemInput{
			{SKU: "A", Qty: intPtr(2)},
			{SKU: "B", Qty: intPtr(5)},
			{SKU: "C", Qty: intPtr(2)},
		},
	})

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "INSUFFICIENT_STOCK", appErr.Code)
	require.Equal(t, "B", appErr.SKU)
	require.NotNil(t, appErr.Available)
	require.Equal(t, 1, *appErr.Available)

	require.Equal(t, 10, testdb.Int(
		t, pool, `SELECT stock FROM products WHERE sku = 'A'`,
	))
	require.Equal(t, 1, testdb.Int(
		t, pool, `SELECT stock FROM products WHERE sku = 'B'`,
	))
	require.Equal(t, 10, testdb.Int(
		t, pool, `SELECT stock FROM products WHERE sku = 'C'`,
	))
	require.Zero(t, testdb.Int(t, pool, `SELECT count(*) FROM sales_orders`))
	require.Zero(t, testdb.Int(t, pool, `SELECT count(*) FROM sales_order_items`))
	require.Zero(t, testdb.Int(t, pool, `SELECT count(*) FROM stock_movements`))
}

func TestCreate_UnknownSKU(t *testing.T) {
	_, pool := apitest.Start(t)
	service := order.NewService(pool)

	testdb.InsertProduct(t, pool, "A", 100, 10)

	_, err := service.Create(t.Context(), order.CreateInput{
		Customer: "Budi",
		Items: []order.ItemInput{
			{SKU: "A", Qty: intPtr(1)},
			{SKU: "NOPE", Qty: intPtr(1)},
		},
	})

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "VALIDATION_FAILED", appErr.Code)
	require.Equal(t, []apperr.Detail{
		{
			Field:   "items[1].sku",
			Message: `unknown sku "NOPE"`,
		},
	}, appErr.Details)
	require.Zero(t, testdb.Int(t, pool, `SELECT count(*) FROM sales_orders`))
	require.Equal(t, 10, testdb.Int(
		t, pool, `SELECT stock FROM products WHERE sku = 'A'`,
	))
}

func TestCreate_InactiveSKU(t *testing.T) {
	_, pool := apitest.Start(t)
	service := order.NewService(pool)

	id := testdb.InsertProduct(t, pool, "A", 100, 10)
	testdb.Exec(t, pool,
		`UPDATE products SET is_active = false WHERE id = $1`,
		id,
	)

	_, err := service.Create(t.Context(), order.CreateInput{
		Customer: "Budi",
		Items: []order.ItemInput{
			{SKU: "A", Qty: intPtr(1)},
		},
	})

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "VALIDATION_FAILED", appErr.Code)
	require.Equal(t, []apperr.Detail{
		{
			Field:   "items[0].sku",
			Message: `sku "A" is inactive`,
		},
	}, appErr.Details)
}

func TestCreate_PriceComesFromDB(t *testing.T) {
	_, pool := apitest.Start(t)
	service := order.NewService(pool)

	testdb.InsertProduct(t, pool, "A", 1000, 10)
	testdb.Exec(t, pool,
		`UPDATE products SET price = 1234 WHERE sku = 'A'`,
	)

	created, err := service.Create(t.Context(), order.CreateInput{
		Customer: "Budi",
		Items: []order.ItemInput{
			{SKU: "A", Qty: intPtr(2)},
		},
	})
	require.NoError(t, err)

	require.Equal(t, 1234, created.Items[0].UnitPrice)
	require.Equal(t, int64(2468), created.Items[0].LineTotal)
}

func TestGet_NotFound(t *testing.T) {
	_, pool := apitest.Start(t)
	service := order.NewService(pool)

	_, err := service.Get(t.Context(), 999)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "NOT_FOUND", appErr.Code)
}
