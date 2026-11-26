package testdb

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/db"
	"stockflow/backend/migrations"
)

var (
	resetOnce sync.Once
	resetErr  error
)

func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — run: docker compose run --rm backend-test")
	}

	ctx := context.Background()

	pool, err := db.Connect(ctx, url, 20)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	resetOnce.Do(func() {
		resetErr = resetSchema(ctx, pool)
	})
	require.NoError(t, resetErr)

	_, err = pool.Exec(
		ctx,
		`TRUNCATE stock_movements, sales_order_items, sales_orders, products RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err)

	return pool
}

func resetSchema(ctx context.Context, pool *pgxpool.Pool) error {
	// Tanpa query arguments, pgx dapat menjalankan kedua statement ini sekaligus.
	_, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`)
	if err != nil {
		return fmt.Errorf("reset public schema: %w", err)
	}

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		sql, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("run migration %s: %w", name, err)
		}
	}

	return nil
}

func InsertProduct(
	t *testing.T,
	pool *pgxpool.Pool,
	sku string,
	price int,
	stock int,
) int {
	t.Helper()

	var id int
	err := pool.QueryRow(
		context.Background(),
		`INSERT INTO products (sku, name, price, stock)
         VALUES ($1, $1, $2, $3)
         RETURNING id`,
		sku,
		price,
		stock,
	).Scan(&id)
	require.NoError(t, err)

	return id
}

func Exec(
	t *testing.T,
	pool *pgxpool.Pool,
	query string,
	args ...any,
) {
	t.Helper()

	_, err := pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
}

func Int(
	t *testing.T,
	pool *pgxpool.Pool,
	query string,
	args ...any,
) int {
	t.Helper()

	var value int
	err := pool.QueryRow(context.Background(), query, args...).Scan(&value)
	require.NoError(t, err)

	return value
}
