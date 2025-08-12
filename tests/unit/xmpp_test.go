package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"veilsupport/internal/config"
	"veilsupport/internal/xmpp"
)

func TestXMPPManager_ConnectAdmin(t *testing.T) {
	cfg := &config.Config{
		XMPP: struct {
			Server   string `mapstructure:"server"`
			Admin    string `mapstructure:"admin"`
			Password string `mapstructure:"password"`
			Domain   string `mapstructure:"domain"`
		}{
			Server:   "localhost:5222",
			Admin:    "admin@test.local",
			Password: "testpass",
			Domain:   "test.local",
		},
	}

	manager := xmpp.NewXMPPManager(cfg)

	// Test connection - this will be mocked in unit tests
	err := manager.ConnectAdmin()
	assert.NoError(t, err)
	assert.True(t, manager.IsAdminConnected())
}

func TestXMPPManager_CreateUserSession(t *testing.T) {
	cfg := &config.Config{
		XMPP: struct {
			Server   string `mapstructure:"server"`
			Admin    string `mapstructure:"admin"`
			Password string `mapstructure:"password"`
			Domain   string `mapstructure:"domain"`
		}{
			Domain: "test.local",
		},
	}

	manager := xmpp.NewXMPPManager(cfg)
	
	// Test user session creation
	userJID := "user123@test.local"
	client, err := manager.CreateUserSession(userJID)
	
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.True(t, manager.HasUserSession(userJID))
}

func TestXMPPManager_SendMessage(t *testing.T) {
	cfg := &config.Config{
		XMPP: struct {
			Server   string `mapstructure:"server"`
			Admin    string `mapstructure:"admin"`
			Password string `mapstructure:"password"`
			Domain   string `mapstructure:"domain"`
		}{
			Admin:  "admin@test.local",
			Domain: "test.local",
		},
	}

	manager := xmpp.NewXMPPManager(cfg)
	
	// First connect admin to test admin sending
	err := manager.ConnectAdmin()
	require.NoError(t, err)
	
	// Test admin sending message
	from := "admin@test.local"
	to := "user123@test.local"
	message := "Hello, I need help!"
	
	err = manager.SendMessage(from, to, message)
	assert.NoError(t, err)
	
	// Create user session and test user sending
	userJID := "user123@test.local"
	_, err = manager.CreateUserSession(userJID)
	require.NoError(t, err)
	
	err = manager.SendMessage(userJID, "admin@test.local", "User reply")
	assert.NoError(t, err)
}

func TestXMPPManager_HandleIncoming(t *testing.T) {
	cfg := &config.Config{
		XMPP: struct {
			Server   string `mapstructure:"server"`
			Admin    string `mapstructure:"admin"`
			Password string `mapstructure:"password"`
			Domain   string `mapstructure:"domain"`
		}{
			Domain: "test.local",
		},
	}

	manager := xmpp.NewXMPPManager(cfg)
	
	// Test incoming message handling
	messagesChan := manager.HandleIncoming()
	assert.NotNil(t, messagesChan)
	
	// Start listening in background
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	go manager.StartListening(ctx)
	
	// Channel should be ready to receive
	select {
	case <-messagesChan:
		// Message received (in real implementation)
	case <-ctx.Done():
		// Timeout is expected in unit test
	}
}

func TestXMPPManager_Disconnect(t *testing.T) {
	cfg := &config.Config{
		XMPP: struct {
			Server   string `mapstructure:"server"`
			Admin    string `mapstructure:"admin"`
			Password string `mapstructure:"password"`
			Domain   string `mapstructure:"domain"`
		}{
			Domain: "test.local",
		},
	}

	manager := xmpp.NewXMPPManager(cfg)
	
	// Test disconnection
	userJID := "user123@test.local"
	_, err := manager.CreateUserSession(userJID)
	require.NoError(t, err)
	
	// Disconnect user
	err = manager.DisconnectUser(userJID)
	assert.NoError(t, err)
	assert.False(t, manager.HasUserSession(userJID))
	
	// Disconnect admin
	err = manager.DisconnectAdmin()
	assert.NoError(t, err)
	assert.False(t, manager.IsAdminConnected())
}