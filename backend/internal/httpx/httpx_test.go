package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apperr"
)

func TestWriteError_InsufficientStockKeepsZeroAvailable(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	WriteError(
		rec,
		req,
		apperr.InsufficientStock("SKU-1", 0),
	)

	require.Equal(t, http.StatusConflict, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	require.Equal(t, map[string]any{
		"error": map[string]any{
			"code":      "INSUFFICIENT_STOCK",
			"message":   "insufficient stock",
			"sku":       "SKU-1",
			"available": float64(0),
		},
	}, body)
}

func TestWriteError_OmitsEmptyFields(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	WriteError(rec, req, apperr.NotFound("product not found"))

	require.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body["error"], &envelope))

	require.NotContains(t, envelope, "available")
	require.NotContains(t, envelope, "sku")
	require.NotContains(t, envelope, "details")
}

func TestWriteError_UnknownErrorIs500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	WriteError(rec, req, errors.New("boom"))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":"INTERNAL"`)
	require.NotContains(t, rec.Body.String(), "boom")
}

func TestDecodeJSON_Rejects(t *testing.T) {
	bodies := []string{
		`{"qty":1,"unit_price":5}`,
		`{"qty":1.5}`,
		`{"qty":"1"}`,
		`{"qty":1}{"qty":2}`,
		`""`,
		`{`,
	}

	for _, body := range bodies {
		t.Run(body, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(body),
			)
			rec := httptest.NewRecorder()

			dst := struct {
				Qty int `json:"qty"`
			}{}

			err := DecodeJSON(rec, req, &dst)

			var appErr *apperr.Error
			require.ErrorAs(t, err, &appErr)
			require.Equal(t, "INVALID_JSON", appErr.Code)
		})
	}
}

func TestJSONErrors_Rewrites404And405(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/x", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"ok": "yes"})
	})

	handler := JSONErrors(mux)

	t.Run("404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/nope", nil)

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		require.Contains(t, rec.Body.String(), `"code":"NOT_FOUND"`)
	})

	t.Run("405", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/x", nil)

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		require.Contains(t, rec.Body.String(), `"code":"METHOD_NOT_ALLOWED"`)
	})
}

func TestRecover_PanicBecomes500Envelope(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":"INTERNAL"`)
	require.NotContains(t, rec.Body.String(), "boom")
}

func TestPathID(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /p/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := PathID(r, "id")
		if err != nil {
			WriteError(w, r, err)
			return
		}

		WriteJSON(w, http.StatusOK, map[string]int{"id": id})
	})

	tests := []struct {
		path       string
		statusCode int
	}{
		{"/p/12", http.StatusOK},
		{"/p/abc", http.StatusNotFound},
		{"/p/0", http.StatusNotFound},
		{"/p/-1", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			mux.ServeHTTP(rec, req)

			require.Equal(t, tt.statusCode, rec.Code)
		})
	}
}
