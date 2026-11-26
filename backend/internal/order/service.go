package order

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stockflow/backend/internal/apperr"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Order, error) {
	indexes := make([]int, len(in.Items))
	for i := range indexes {
		indexes[i] = i
	}

	slices.SortFunc(indexes, func(a, b int) int {
		return strings.Compare(in.Items[a].SKU, in.Items[b].SKU)
	})

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	skus := make([]string, len(in.Items))
	for i, item := range in.Items {
		skus[i] = item.SKU
	}

	type skuState struct {
		SKU      string
		IsActive bool
	}

	rows, err := tx.Query(
		ctx,
		`SELECT sku, is_active FROM products WHERE sku = ANY($1)`,
		skus,
	)
	if err != nil {
		return Order{}, err
	}

	states, err := pgx.CollectRows(rows, pgx.RowToStructByPos[skuState])
	if err != nil {
		return Order{}, err
	}

	active := make(map[string]bool, len(states))
	for _, state := range states {
		active[state.SKU] = state.IsActive
	}

	var details []apperr.Detail
	for i, item := range in.Items {
		isActive, found := active[item.SKU]

		switch {
		case !found:
			details = append(details, apperr.Detail{
				Field:   fmt.Sprintf("items[%d].sku", i),
				Message: fmt.Sprintf("unknown sku %q", item.SKU),
			})
		case !isActive:
			details = append(details, apperr.Detail{
				Field:   fmt.Sprintf("items[%d].sku", i),
				Message: fmt.Sprintf("sku %q is inactive", item.SKU),
			})
		}
	}

	if len(details) > 0 {
		return Order{}, apperr.Validation(details)
	}

	type line struct {
		ProductID int
		Qty       int
		UnitPrice int
		SKU       string
	}

	lines := make([]line, 0, len(indexes))
	var total int64

	for _, i := range indexes {
		item := in.Items[i]

		var l line
		err := tx.QueryRow(ctx, `
            UPDATE products
            SET stock = stock - $2
            WHERE sku = $1
              AND is_active
              AND stock >= $2
            RETURNING id, price`,
			item.SKU,
			*item.Qty,
		).Scan(&l.ProductID, &l.UnitPrice)

		if errors.Is(err, pgx.ErrNoRows) {
			var available int
			var isActive bool

			err = tx.QueryRow(
				ctx,
				`SELECT stock, is_active FROM products WHERE sku = $1`,
				item.SKU,
			).Scan(&available, &isActive)

			if err != nil {
				return Order{}, err
			}
			if !isActive {
				return Order{}, apperr.Validation([]apperr.Detail{
					{
						Field:   fmt.Sprintf("items[%d].sku", i),
						Message: fmt.Sprintf("sku %q is inactive", item.SKU),
					},
				})
			}

			return Order{}, apperr.InsufficientStock(item.SKU, available)
		}
		if err != nil {
			return Order{}, err
		}

		l.Qty = *item.Qty
		l.SKU = item.SKU
		total += int64(l.Qty) * int64(l.UnitPrice)
		lines = append(lines, l)
	}

	var orderID int
	err = tx.QueryRow(ctx, `
        INSERT INTO sales_orders (customer, total)
        VALUES ($1, $2)
        RETURNING id`,
		in.Customer,
		total,
	).Scan(&orderID)
	if err != nil {
		return Order{}, err
	}

	for _, l := range lines {
		_, err = tx.Exec(ctx, `
            INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
            VALUES ($1, $2, $3, $4)`,
			orderID,
			l.SKU,
			l.Qty,
			l.UnitPrice,
		)
		if err != nil {
			return Order{}, err
		}

		_, err = tx.Exec(ctx, `
            INSERT INTO stock_movements
                (product_id, type, qty, note, order_id)
            VALUES ($1, 'out', $2, $3, $4)`,
			l.ProductID,
			l.Qty,
			fmt.Sprintf("order #%d", orderID),
			orderID,
		)
		if err != nil {
			return Order{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}

	return s.Get(ctx, orderID)
}

func (s *Service) Get(ctx context.Context, id int) (Order, error) {
	var result Order

	err := s.pool.QueryRow(ctx, `
        SELECT id, customer, status, total, created_at, cancelled_at
        FROM sales_orders
        WHERE id = $1`,
		id,
	).Scan(
		&result.ID,
		&result.Customer,
		&result.Status,
		&result.Total,
		&result.CreatedAt,
		&result.CancelledAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, apperr.NotFound("order not found")
	}
	if err != nil {
		return Order{}, err
	}

	rows, err := s.pool.Query(ctx, `
        SELECT i.id,
               i.sku,
               p.name,
               i.qty,
               i.unit_price,
               i.qty::bigint * i.unit_price
        FROM sales_order_items i
        JOIN products p ON p.sku = i.sku
        WHERE i.order_id = $1
        ORDER BY i.id`,
		id,
	)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()

	result.Items = make([]Item, 0)
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.ID,
			&item.SKU,
			&item.Name,
			&item.Qty,
			&item.UnitPrice,
			&item.LineTotal,
		)
		if err != nil {
			return Order{}, err
		}
		result.Items = append(result.Items, item)
	}

	if err := rows.Err(); err != nil {
		return Order{}, err
	}

	return result, nil
}

func (s *Service) List(ctx context.Context, page, limit int) ([]Summary, int, error) {
	var total int
	err := s.pool.QueryRow(
		ctx,
		`SELECT count(*) FROM sales_orders`,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := s.pool.Query(ctx, `
		SELECT o.id,
		       o.customer,
		       o.status,
		       o.total,
		       o.created_at,
		       o.cancelled_at,
		       (
		         SELECT count(*)
		         FROM sales_order_items i
		         WHERE i.order_id = o.id
		       )
		FROM sales_orders o
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	summaries := make([]Summary, 0)
	for rows.Next() {
		var summary Summary
		err := rows.Scan(
			&summary.ID,
			&summary.Customer,
			&summary.Status,
			&summary.Total,
			&summary.CreatedAt,
			&summary.CancelledAt,
			&summary.ItemCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan order summary: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate order summaries: %w", err)
	}

	return summaries, total, nil
}
