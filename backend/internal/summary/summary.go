package summary

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"stockflow/backend/internal/httpx"
)

type Summary struct {
	TotalProducts   int   `json:"total_products"`
	ActiveProducts  int   `json:"active_products"`
	TotalStockValue int64 `json:"total_stock_value"`
	LowStockCount   int   `json:"low_stock_count"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/summary", httpx.Handle(h.get))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	var result Summary

	err := h.pool.QueryRow(r.Context(), `
		SELECT count(*),
		       count(*) FILTER (WHERE is_active),
		       COALESCE(SUM(stock::bigint * price) FILTER (WHERE is_active), 0),
		       count(*) FILTER (WHERE is_active AND stock <= 10)
		FROM products`,
	).Scan(
		&result.TotalProducts,
		&result.ActiveProducts,
		&result.TotalStockValue,
		&result.LowStockCount,
	)
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, result)
	return nil
}
