package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"veilsupport/internal/config"
	"veilsupport/internal/database"
	"veilsupport/internal/xmpp"
	"veilsupport/tests/testhelpers"
)

func TestXMPPService_NewService(t *testing.T) {
	// Setup test database
	pool := testhelpers.SetupTestDB(t)
	defer pool.Close()
	
	dbService := database.NewService(pool)
	
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
	
	// Create XMPP service
	service := xmpp.NewService(dbService, nil, cfg)
	
	// Assert service is created properly
	assert.NotNil(t, service)
	assert.NotNil(t, service.GetChatService())
	assert.NotNil(t, service.GetXMPPManager())
}

func TestXMPPService_Start(t *testing.T) {
	// Setup test database
	pool := testhelpers.SetupTestDB(t)
	defer pool.Close()
	
	dbService := database.NewService(pool)
	
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
	
	// Create XMPP service
	service := xmpp.NewService(dbService, nil, cfg)
	
	// Test starting the service
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	err := service.Start(ctx)
	assert.NoError(t, err)
	
	// Verify admin is connected
	assert.True(t, service.GetXMPPManager().IsAdminConnected())
	
	// Stop the service
	service.Stop()
	
	// Verify admin is disconnected
	assert.False(t, service.GetXMPPManager().IsAdminConnected())
}

type MockWSManager struct {
	mock.Mock
}

func (m *MockWSManager) SendToUser(userEmail, message string) error {
	args := m.Called(userEmail, message)
	return args.Error(0)
}

func TestXMPPService_SendUserMessage(t *testing.T) {
	// Setup test database
	pool := testhelpers.SetupTestDB(t)
	defer pool.Close()
	
	dbService := database.NewService(pool)
	
	// Create test user first
	testUser, err := dbService.CreateUser(context.Background(), "test@example.com", "hashedpass", "user_test_example_com@test.local")
	assert.NoError(t, err)
	assert.NotNil(t, testUser)
	
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
	
	mockWS := &MockWSManager{}
	
	// Create XMPP service
	service := xmpp.NewService(dbService, mockWS, cfg)
	
	// Send user message
	userEmail := "test@example.com"
	message := "Hello, I need help!"
	
	err = service.SendUserMessage(userEmail, message)
	assert.NoError(t, err)
}

func TestXMPPService_SendAdminMessage(t *testing.T) {
	// Setup test database
	pool := testhelpers.SetupTestDB(t)
	defer pool.Close()
	
	dbService := database.NewService(pool)
	
	// Create test user first
	testUser, err := dbService.CreateUser(context.Background(), "test@example.com", "hashedpass", "user_test_example_com@test.local")
	assert.NoError(t, err)
	assert.NotNil(t, testUser)
	
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
	
	mockWS := &MockWSManager{}
	mockWS.On("SendToUser", "test@example.com", "Thank you for contacting us!").Return(nil)
	
	// Create XMPP service
	service := xmpp.NewService(dbService, mockWS, cfg)
	
	// Send admin message
	targetUser := "test@example.com"
	message := "Thank you for contacting us!"
	
	err = service.SendAdminMessage(targetUser, message)
	assert.NoError(t, err)
	
	mockWS.AssertExpectations(t)
}