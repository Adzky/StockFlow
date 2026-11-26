package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"stockflow/backend/internal/apperr"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperr.Error

	if errors.As(err, &appErr) {
		WriteJSON(w, appErr.Status, map[string]any{
			"error": appErr,
		})
		return
	}

	slog.Error(
		"internal server error",
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	)

	WriteJSON(w, http.StatusInternalServerError, map[string]any{
		"error": apperr.New(
			http.StatusInternalServerError,
			"INTERNAL",
			"internal server error",
		),
	})
}

func Handle(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, r, err)
		}
	}
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return apperr.InvalidJSON("invalid JSON")
	}

	var extra struct{}

	if err := dec.Decode(&extra); err != io.EOF {
		return apperr.InvalidJSON("invalid JSON")
	}

	return nil
}

func PathID(r *http.Request, name string) (int, error) {
	idText := r.PathValue(name)

	id, err := strconv.Atoi(idText)
	if err != nil || id < 1 {
		return 0, apperr.NotFound("not found")
	}

	return id, nil
}
