package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apitest"
)

func TestHealth(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/health",
		"",
	)

	require.Equal(t, http.StatusOK, status)
	require.JSONEq(t, `{"status":"ok"}`, string(body))
}

func TestUnknownRouteIsJSON404(t *testing.T) {
	srv, _ := apitest.Start(t)

	status, body := apitest.Do(
		t,
		srv,
		http.MethodGet,
		"/api/nope",
		"",
	)

	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, "NOT_FOUND", apitest.Err(t, body).Code)
}
