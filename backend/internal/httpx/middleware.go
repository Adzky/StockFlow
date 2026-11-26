package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"stockflow/backend/internal/apperr"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error(
					"panic recovered",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
				)

				WriteJSON(w, http.StatusInternalServerError, map[string]any{
					"error": apperr.New(
						http.StatusInternalServerError,
						"INTERNAL",
						"internal server error",
					),
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info(
				"request",
				"method", r.Method,
				"path", r.URL.Path,
			)

			next.ServeHTTP(w, r)
		})
	}
}

type jsonErrorWriter struct {
	http.ResponseWriter
	swallow bool
}

func (w *jsonErrorWriter) WriteHeader(code int) {
	contentType := w.Header().Get("Content-Type")
	isPlain := !strings.HasPrefix(contentType, "application/json")

	if isPlain &&
		(code == http.StatusNotFound ||
			code == http.StatusMethodNotAllowed) {
		w.swallow = true
		w.Header().Set("Content-Type", "application/json")

		err := apperr.NotFound("route not found")
		if code == http.StatusMethodNotAllowed {
			err = apperr.New(
				http.StatusMethodNotAllowed,
				"METHOD_NOT_ALLOWED",
				"method not allowed",
			)
		}

		w.ResponseWriter.WriteHeader(code)

		_ = json.NewEncoder(w.ResponseWriter).Encode(map[string]any{
			"error": err,
		})
		return
	}

	w.ResponseWriter.WriteHeader(code)
}

func (w *jsonErrorWriter) Write(body []byte) (int, error) {
	if w.swallow {
		return len(body), nil
	}

	return w.ResponseWriter.Write(body)
}

func JSONErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &jsonErrorWriter{
			ResponseWriter: w,
		}

		next.ServeHTTP(writer, r)
	})
}
