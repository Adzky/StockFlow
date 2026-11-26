package server

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"stockflow/backend/internal/apperr"
	"stockflow/backend/internal/httpx"
)

func New(pool *pgxpool.Pool, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			httpx.WriteError(
				w,
				r,
				apperr.New(
					http.StatusServiceUnavailable,
					"UNAVAILABLE",
					"database unavailable",
				),
			)
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	return httpx.Logger(log)(
		httpx.Recover(
			httpx.JSONErrors(mux),
		),
	)
}
