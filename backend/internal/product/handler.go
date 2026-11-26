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
