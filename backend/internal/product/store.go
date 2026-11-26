package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"stockflow/backend/internal/apperr"
	"stockflow/backend/internal/db"
)

const productSelect = `
SELECT p.id, p.sku, p.name, p.price, p.stock, p.is_active, p.created_at, lm.created_at
FROM products p
LEFT JOIN LATERAL (
	SELECT m.created_at FROM stock_movements m
	WHERE m.product_id = p.id
	ORDER BY m.created_at DESC, m.id DESC
	LIMIT 1
) lm ON true`

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, in CreateInput) (Product, error) {
	stock := 0
	if in.Stock != nil {
		stock = *in.Stock
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Product{}, fmt.Errorf("begin create product transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int
	err = tx.QueryRow(
		ctx,
		`INSERT INTO products (sku, name, price, stock, is_active)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		in.SKU,
		in.Name,
		*in.Price,
		stock,
		isActive,
	).Scan(&id)
	if db.IsPgCode(err, "23505") {
		return Product{}, apperr.Conflict(
			"SKU_ALREADY_EXISTS",
			fmt.Sprintf("sku %q already exists", in.SKU),
		)
	}
	if err != nil {
		return Product{}, fmt.Errorf("insert product: %w", err)
	}

	if stock > 0 {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO stock_movements (product_id, type, qty, note)
			 VALUES ($1, $2, $3, $4)`,
			id,
			MovementIn,
			stock,
			"initial stock",
		)
		if err != nil {
			return Product{}, fmt.Errorf("insert initial stock movement: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Product{}, fmt.Errorf("commit create product transaction: %w", err)
	}

	return s.Get(ctx, id)
}

func (s *Store) Get(ctx context.Context, id int) (Product, error) {
	product, err := scanProduct(
		s.pool.QueryRow(ctx, productSelect+`
WHERE p.id = $1`, id),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, apperr.NotFound("product not found")
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product: %w", err)
	}

	return product, nil
}

func (s *Store) Update(
	ctx context.Context,
	id int,
	in UpdateInput,
) (Product, error) {
	var updatedID int
	err := s.pool.QueryRow(
		ctx,
		`UPDATE products
		 SET name = $2, price = $3, is_active = $4
		 WHERE id = $1
		 RETURNING id`,
		id,
		*in.Name,
		*in.Price,
		*in.IsActive,
	).Scan(&updatedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, apperr.NotFound("product not found")
	}
	if err != nil {
		return Product{}, fmt.Errorf("update product: %w", err)
	}

	return s.Get(ctx, updatedID)
}

func scanProduct(row pgx.Row) (Product, error) {
	var product Product
	var lastMovementAt pgtype.Timestamptz

	err := row.Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Price,
		&product.Stock,
		&product.IsActive,
		&product.CreatedAt,
		&lastMovementAt,
	)
	if err != nil {
		return Product{}, err
	}

	if lastMovementAt.Valid {
		product.LastMovementAt = &lastMovementAt.Time
	}

	return product, nil
}
