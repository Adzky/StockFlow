package order

import (
	"net/http"

	"stockflow/backend/internal/apperr"
	"stockflow/backend/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/orders", httpx.Handle(h.create))
	mux.Handle("GET /api/orders/{id}", httpx.Handle(h.get))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var in CreateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}

	if details := ValidateCreate(&in); len(details) > 0 {
		return apperr.Validation(details)
	}

	created, err := h.service.Create(r.Context(), in)
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusCreated, created)
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}

	result, err := h.service.Get(r.Context(), id)
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, result)
	return nil
}
