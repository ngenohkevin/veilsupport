package testhelpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// SetupTestServer creates a test HTTP server for integration tests
func SetupTestServer(t *testing.T, pool *pgxpool.Pool) *httptest.Server {
	// This will be implemented when we create the main server in later phases
	// For now, return a basic server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := httptest.NewServer(mux)
	
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// PostJSON sends a POST request with JSON payload to a test server
func PostJSON(t *testing.T, url string, payload interface{}) *http.Response {
	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)

	return resp
}

// PostJSONWithAuth sends a POST request with JSON payload and authorization header
func PostJSONWithAuth(t *testing.T, url string, payload interface{}, token string) *http.Response {
	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}