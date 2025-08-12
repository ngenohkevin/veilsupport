package testhelpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// SetupTestServer creates a test HTTP server for integration tests using the actual Gin router
func SetupTestServer(t *testing.T, pool *pgxpool.Pool) *httptest.Server {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create Gin router with same setup as main.go
	router := gin.Default()
	
	// Add the health endpoint exactly as in main.go
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "veilsupport",
			"version": "0.1.0",
		})
	})

	// TODO: Add other routes when implemented in later phases

	server := httptest.NewServer(router)
	
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