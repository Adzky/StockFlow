package apitest

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apperr"
	"stockflow/backend/internal/server"
	"stockflow/backend/internal/testdb"
)

func Start(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()

	pool := testdb.New(t)
	handler := server.New(pool, slog.New(slog.DiscardHandler))
	srv := httptest.NewServer(handler)

	t.Cleanup(srv.Close)

	return srv, pool
}

func Do(
	t *testing.T,
	srv *httptest.Server,
	method string,
	path string,
	body string,
) (int, []byte) {
	t.Helper()

	var requestBody io.Reader
	if body != "" {
		requestBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, srv.URL+path, requestBody)
	if err != nil {
		t.Errorf("create request: %v", err)
		return 0, nil
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("send request: %v", err)
		return 0, nil
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("read response body: %v", err)
		return resp.StatusCode, nil
	}

	return resp.StatusCode, raw
}

func Err(t *testing.T, raw []byte) apperr.Error {
	t.Helper()

	var envelope struct {
		Error apperr.Error `json:"error"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))

	return envelope.Error
}
