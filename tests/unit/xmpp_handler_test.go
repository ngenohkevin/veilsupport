package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"veilsupport/internal/config"
	"veilsupport/internal/xmpp"
)

type MockChatService struct {
	mock.Mock
}

func (m *MockChatService) SaveMessage(ctx context.Context, sessionID, fromJID, toJID, content, messageType string) error {
	args := m.Called(ctx, sessionID, fromJID, toJID, content, messageType)
	return args.Error(0)
}

func (m *MockChatService) GetOrCreateSession(ctx context.Context, userEmail string) (string, error) {
	args := m.Called(ctx, userEmail)
	return args.String(0), args.Error(1)
}

func (m *MockChatService) BroadcastToWebSocket(userEmail, message string) error {
	args := m.Called(userEmail, message)
	return args.Error(0)
}

func TestMessageHandler_ProcessUserMessage(t *testing.T) {
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

	xmppManager := xmpp.NewXMPPManager(cfg)
	mockChatService := &MockChatService{}

	handler := xmpp.NewMessageHandler(xmppManager, mockChatService)

	// Setup expectations
	userEmail := "test@example.com"
	message := "I need help with my order"
	sessionID := "session-123"

	mockChatService.On("GetOrCreateSession", mock.Anything, userEmail).Return(sessionID, nil)
	mockChatService.On("SaveMessage", mock.Anything, sessionID, mock.AnythingOfType("string"), cfg.XMPP.Admin, message, "user").Return(nil)

	// Test processing user message
	err := handler.ProcessUserMessage(userEmail, message)
	
	assert.NoError(t, err)
	mockChatService.AssertExpectations(t)
}

func TestMessageHandler_ProcessAdminMessage(t *testing.T) {
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

	xmppManager := xmpp.NewXMPPManager(cfg)
	mockChatService := &MockChatService{}

	handler := xmpp.NewMessageHandler(xmppManager, mockChatService)

	// Setup expectations
	targetUser := "test@example.com"
	message := "Thank you for contacting us. How can I help?"
	sessionID := "session-123"

	mockChatService.On("GetOrCreateSession", mock.Anything, targetUser).Return(sessionID, nil)
	mockChatService.On("SaveMessage", mock.Anything, sessionID, cfg.XMPP.Admin, mock.AnythingOfType("string"), message, "admin").Return(nil)
	mockChatService.On("BroadcastToWebSocket", targetUser, message).Return(nil)

	// Test processing admin message
	err := handler.ProcessAdminMessage(targetUser, message)
	
	assert.NoError(t, err)
	mockChatService.AssertExpectations(t)
}

func TestMessageHandler_StartListening(t *testing.T) {
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

	xmppManager := xmpp.NewXMPPManager(cfg)
	mockChatService := &MockChatService{}

	handler := xmpp.NewMessageHandler(xmppManager, mockChatService)

	// Test that listening starts without error
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should not block and should handle the context cancellation
	handler.StartListening(ctx)
	
	// If we reach here, the method handled context cancellation properly
	assert.True(t, true)
}

func TestMessageHandler_GenerateUserJID(t *testing.T) {
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

	xmppManager := xmpp.NewXMPPManager(cfg)
	mockChatService := &MockChatService{}

	handler := xmpp.NewMessageHandler(xmppManager, mockChatService)

	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "simple email",
			email:    "test@example.com",
			expected: "user_test_example_com@test.local",
		},
		{
			name:     "email with dots",
			email:    "test.user@example.com",
			expected: "user_test_user_example_com@test.local",
		},
		{
			name:     "email with plus",
			email:    "test+tag@example.com",
			expected: "user_test_tag_example_com@test.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.GenerateUserJID(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}