package unit

import (
	"testing"

	"veilsupport/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Load_WithDefaults(t *testing.T) {
	// Test that configuration loads with default values
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test default values
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, 3600, cfg.JWT.TTL)
}

func TestConfig_ServerDefaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Empty(t, cfg.Server.TorProxy) // Should be empty by default
}

func TestConfig_DatabaseDefaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "veilsupport", cfg.Database.User)
	assert.Equal(t, "veilsupport", cfg.Database.DBName)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
}

func TestConfig_XMPPDefaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.XMPP.Domain)
}

func TestConfig_JWTDefaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 3600, cfg.JWT.TTL) // 1 hour in seconds
}