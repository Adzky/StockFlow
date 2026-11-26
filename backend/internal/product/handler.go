package product

import (
	"net/http"

	"stockflow/backend/internal/apperr"
	"stockflow/backend/internal/httpx"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/products", httpx.Handle(h.create))
	mux.Handle("PUT /api/products/{id}", httpx.Handle(h.update))
	mux.Handle("GET /api/products", httpx.Handle(h.list))
	mux.Handle("DELETE /api/products/{id}", httpx.Handle(h.remove))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var in CreateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}

	if details := ValidateCreate(&in); len(details) > 0 {
		return apperr.Validation(details)
	}

	created, err := h.store.Create(r.Context(), in)
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusCreated, created)
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}

	var in UpdateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}

	if details := ValidateUpdate(&in); len(details) > 0 {
		return apperr.Validation(details)
	}

	updated, err := h.store.Update(r.Context(), id, in)
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, updated)
	return nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	page, err := httpx.QueryInt(r, "page", 1, 1, 1_000_000)
	if err != nil {
		return err
	}

	limit, err := httpx.QueryInt(r, "limit", 20, 1, 100)
	if err != nil {
		return err
	}

	filter := r.URL.Query().Get("filter")
	if filter == "" {
		filter = "all"
	}

	switch filter {
	case "all", "active", "inactive", "low_stock":
	default:
		return apperr.Validation([]apperr.Detail{
			{
				Field:   "filter",
				Message: "must be one of all, active, inactive, low_stock",
			},
		})
	}

	products, total, err := h.store.List(r.Context(), ListParams{
		Search: r.URL.Query().Get("search"),
		Filter: filter,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.Paged[Product]{
		Data:  products,
		Page:  page,
		Limit: limit,
		Total: total,
	})
	return nil
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
