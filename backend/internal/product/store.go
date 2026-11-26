package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

const listWhere = `
WHERE ($1::text = '' OR p.sku ILIKE $1 OR p.name ILIKE $1)
  AND CASE $2::text
        WHEN 'active'    THEN p.is_active
        WHEN 'inactive'  THEN NOT p.is_active
        WHEN 'low_stock' THEN p.is_active AND p.stock <= 10
        ELSE TRUE
      END`

type ListParams struct {
	Search string
	Filter string
	Page   int
	Limit  int
}

func (s *Store) List(
	ctx context.Context,
	p ListParams,
) ([]Product, int, error) {
	pattern := ""
	if search := strings.TrimSpace(p.Search); search != "" {
		pattern = "%" + likeEscaper.Replace(search) + "%"
	}

	var total int
	err := s.pool.QueryRow(
		ctx,
		`SELECT count(*) FROM products p`+listWhere,
		pattern,
		p.Filter,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	offset := (p.Page - 1) * p.Limit
	rows, err := s.pool.Query(
		ctx,
		productSelect+listWhere+`
ORDER BY p.id
LIMIT $3 OFFSET $4`,
		pattern,
		p.Filter,
		p.Limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		item, err := scanProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate products: %w", err)
	}

	return products, total, nil
}

func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(
		ctx,
		`DELETE FROM products p
         WHERE p.id = $1
           AND NOT EXISTS (
               SELECT 1
               FROM stock_movements m
               WHERE m.product_id = p.id
           )`,
		id,
	)
	if db.IsPgCode(err, "23503") {
		return apperr.Conflict(
			"PRODUCT_HAS_HISTORY",
			"product has stock history and cannot be deleted",
		)
	}
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	if tag.RowsAffected() > 0 {
		return nil
	}

	var exists bool
	err = s.pool.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`,
		id,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check product after delete: %w", err)
	}

	if !exists {
		return apperr.NotFound("product not found")
	}

	return apperr.Conflict(
		"PRODUCT_HAS_HISTORY",
		"product has stock history and cannot be deleted",
	)
}

func (s *Store) AdjustStock(
	ctx context.Context,
	id int,
	in AdjustInput,
) (AdjustResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AdjustResult{}, err
	}
	defer tx.Rollback(ctx)

	query := `UPDATE products SET stock = stock + $2
        WHERE id = $1 RETURNING id`

	if in.Type == MovementOut {
		query = `UPDATE products SET stock = stock - $2
            WHERE id = $1 AND stock >= $2 RETURNING id`
	}

	var updatedID int
	err = tx.QueryRow(ctx, query, id, *in.Qty).Scan(&updatedID)
	if errors.Is(err, pgx.ErrNoRows) {
		var available int
		err = tx.QueryRow(
			ctx,
			`SELECT stock FROM products WHERE id = $1`,
			id,
		).Scan(&available)

		if errors.Is(err, pgx.ErrNoRows) {
			return AdjustResult{}, apperr.NotFound("product not found")
		}
		if err != nil {
			return AdjustResult{}, err
		}

		return AdjustResult{}, apperr.InsufficientStock("", available)
	}
	if err != nil {
		return AdjustResult{}, err
	}

	var movement Movement
	err = tx.QueryRow(ctx, `
        INSERT INTO stock_movements (product_id, type, qty, note)
        VALUES ($1, $2, $3, $4)
        RETURNING id, product_id, type, qty, note, order_id, created_at`,
		id,
		in.Type,
		*in.Qty,
		in.Note,
	).Scan(
		&movement.ID,
		&movement.ProductID,
		&movement.Type,
		&movement.Qty,
		&movement.Note,
		&movement.OrderID,
		&movement.CreatedAt,
	)
	if err != nil {
		return AdjustResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AdjustResult{}, err
	}

	product, err := s.Get(ctx, id)
	if err != nil {
		return AdjustResult{}, err
	}

	return AdjustResult{
		Product:  product,
		Movement: movement,
	}, nil
}

func (s *Store) Movements(ctx context.Context, productID int) ([]Movement, error) {
	var exists bool
	err := s.pool.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`,
		productID,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check product movements: %w", err)
	}
	if !exists {
		return nil, apperr.NotFound("product not found")
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, product_id, type, qty, note, order_id, created_at
		FROM stock_movements
		WHERE product_id = $1
		ORDER BY created_at DESC, id DESC`,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf("list stock movements: %w", err)
	}
	defer rows.Close()

	movements := make([]Movement, 0)
	for rows.Next() {
		var movement Movement
		err := rows.Scan(
			&movement.ID,
			&movement.ProductID,
			&movement.Type,
			&movement.Qty,
			&movement.Note,
			&movement.OrderID,
			&movement.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stock movement: %w", err)
		}
		movements = append(movements, movement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock movements: %w", err)
	}

	return movements, nil
}
