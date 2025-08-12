package integration

import (
	"net/http"
	"testing"

	"veilsupport/tests/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	// Setup test database (even though health endpoint doesn't use it)
	pool := testhelpers.SetupTestDB(t)
	
	// Setup test server
	server := testhelpers.SetupTestServer(t, pool)
	defer server.Close()

	// Test health endpoint
	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestHealthEndpoint_ReturnsCorrectResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testhelpers.SetupTestDB(t)
	server := testhelpers.SetupTestServer(t, pool)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}