package integration

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

type MockWSManager struct {
	mock.Mock
	receivedMessages []WSMessage
}

type WSMessage struct {
	UserEmail string
	Message   string
}

func (m *MockWSManager) SendToUser(userEmail, message string) error {
	args := m.Called(userEmail, message)
	m.receivedMessages = append(m.receivedMessages, WSMessage{
		UserEmail: userEmail,
		Message:   message,
	})
	return args.Error(0)
}

func (m *MockWSManager) GetReceivedMessages() []WSMessage {
	return m.receivedMessages
}

func TestXMPPMessageFlowEndToEnd(t *testing.T) {
	// Setup test database
	pool := testhelpers.SetupTestDB(t)
	defer pool.Close()
	
	dbService := database.NewService(pool)
	
	// Create test configuration
	cfg := &config.Config{
		XMPP: struct {
			Server   string `mapstructure:"server"`
			Admin    string `mapstructure:"admin"`
			Password string `mapstructure:"password"`
			Domain   string `mapstructure:"domain"`
		}{
			Server:   "localhost:5222",
			Admin:    "admin@support.local",
			Password: "adminpass",
			Domain:   "support.local",
		},
	}
	
	// Create mock WebSocket manager
	mockWS := &MockWSManager{}
	mockWS.On("SendToUser", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	
	// Create XMPP service
	xmppService := xmpp.NewService(dbService, mockWS, cfg)
	
	// Test data
	userEmail := "customer@example.com"
	userMessage := "Hello, I'm having trouble with my order #12345"
	adminResponse := "Hi! I'd be happy to help you with your order. Let me look that up for you."
	
	// Step 1: Create a test user
	userJID := "user_customer_example_com@support.local"
	testUser, err := dbService.CreateUser(context.Background(), userEmail, "hashedpass", userJID)
	assert.NoError(t, err)
	assert.NotNil(t, testUser)
	
	// Step 2: Start XMPP service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = xmppService.Start(ctx)
	assert.NoError(t, err)
	
	// Verify admin is connected
	assert.True(t, xmppService.GetXMPPManager().IsAdminConnected())
	
	// Step 3: Send user message (simulates user typing in web interface)
	err = xmppService.SendUserMessage(userEmail, userMessage)
	assert.NoError(t, err)
	
	// Step 4: Verify user message was saved to database
	chatService := xmppService.GetChatService()
	messageHistory, err := chatService.GetMessageHistory(context.Background(), userEmail, 10)
	assert.NoError(t, err)
	assert.Len(t, messageHistory, 1)
	assert.Equal(t, userMessage, messageHistory[0].Content)
	assert.Equal(t, "user", messageHistory[0].MessageType)
	
	// Step 5: Send admin response (simulates admin replying via XMPP client)
	err = xmppService.SendAdminMessage(userEmail, adminResponse)
	assert.NoError(t, err)
	
	// Step 6: Verify admin message was saved to database
	messageHistory, err = chatService.GetMessageHistory(context.Background(), userEmail, 10)
	assert.NoError(t, err)
	assert.Len(t, messageHistory, 2)
	
	// Find admin message in history (order might vary)
	var adminMsg *string
	for _, msg := range messageHistory {
		if msg.MessageType == "admin" {
			adminMsg = &msg.Content
			break
		}
	}
	assert.NotNil(t, adminMsg)
	assert.Equal(t, adminResponse, *adminMsg)
	
	// Step 7: Verify WebSocket broadcast was called for admin message
	receivedMessages := mockWS.GetReceivedMessages()
	assert.Len(t, receivedMessages, 1)
	assert.Equal(t, userEmail, receivedMessages[0].UserEmail)
	assert.Equal(t, adminResponse, receivedMessages[0].Message)
	
	// Step 8: Verify session was created and is active
	activeSession, err := dbService.GetActiveSessionByUserID(context.Background(), testUser.ID)
	assert.NoError(t, err)
	assert.Equal(t, "active", activeSession.Status)
	assert.Equal(t, testUser.ID, activeSession.UserID)
	
	// Step 9: Test message history retrieval by session
	sessionHistory, err := chatService.GetSessionHistory(context.Background(), activeSession.ID.String())
	assert.NoError(t, err)
	assert.Len(t, sessionHistory, 2)
	
	// Verify message ordering (should be chronological)
	assert.Equal(t, "user", sessionHistory[0].MessageType)
	assert.Equal(t, userMessage, sessionHistory[0].Content)
	assert.Equal(t, "admin", sessionHistory[1].MessageType)
	assert.Equal(t, adminResponse, sessionHistory[1].Content)
	
	// Step 10: Test session closure
	err = chatService.CloseSession(context.Background(), activeSession.ID.String())
	assert.NoError(t, err)
	
	// Verify session is now closed
	_, err = dbService.GetActiveSessionByUserID(context.Background(), testUser.ID)
	assert.Error(t, err) // Should not find active session
	
	// Step 11: Stop XMPP service
	xmppService.Stop()
	
	// Verify admin is disconnected
	assert.False(t, xmppService.GetXMPPManager().IsAdminConnected())
	
	// Verify all mock expectations were met
	mockWS.AssertExpectations(t)
}

func TestXMPPMessageFlowMultipleUsers(t *testing.T) {
	// Setup test database
	pool := testhelpers.SetupTestDB(t)
	defer pool.Close()
	
	dbService := database.NewService(pool)
	
	// Create test configuration
	cfg := &config.Config{
		XMPP: struct {
			Server   string `mapstructure:"server"`
			Admin    string `mapstructure:"admin"`
			Password string `mapstructure:"password"`
			Domain   string `mapstructure:"domain"`
		}{
			Server:   "localhost:5222",
			Admin:    "admin@support.local",
			Password: "adminpass",
			Domain:   "support.local",
		},
	}
	
	// Create mock WebSocket manager
	mockWS := &MockWSManager{}
	mockWS.On("SendToUser", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	
	// Create XMPP service
	xmppService := xmpp.NewService(dbService, mockWS, cfg)
	
	// Create multiple test users
	user1Email := "customer1@example.com"
	user2Email := "customer2@example.com"
	
	user1, err := dbService.CreateUser(context.Background(), user1Email, "pass1", "user_customer1_example_com@support.local")
	assert.NoError(t, err)
	
	user2, err := dbService.CreateUser(context.Background(), user2Email, "pass2", "user_customer2_example_com@support.local")
	assert.NoError(t, err)
	
	// Start XMPP service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = xmppService.Start(ctx)
	assert.NoError(t, err)
	
	// Test concurrent user messages
	err = xmppService.SendUserMessage(user1Email, "User 1 needs help with billing")
	assert.NoError(t, err)
	
	err = xmppService.SendUserMessage(user2Email, "User 2 has a technical question")
	assert.NoError(t, err)
	
	// Send responses to both users
	err = xmppService.SendAdminMessage(user1Email, "I'll help you with billing")
	assert.NoError(t, err)
	
	err = xmppService.SendAdminMessage(user2Email, "Let me assist with your technical question")
	assert.NoError(t, err)
	
	// Verify each user has their own message history
	chatService := xmppService.GetChatService()
	
	user1History, err := chatService.GetMessageHistory(context.Background(), user1Email, 10)
	assert.NoError(t, err)
	assert.Len(t, user1History, 2)
	
	user2History, err := chatService.GetMessageHistory(context.Background(), user2Email, 10)
	assert.NoError(t, err)
	assert.Len(t, user2History, 2)
	
	// Verify each user has their own active session
	session1, err := dbService.GetActiveSessionByUserID(context.Background(), user1.ID)
	assert.NoError(t, err)
	assert.Equal(t, user1.ID, session1.UserID)
	
	session2, err := dbService.GetActiveSessionByUserID(context.Background(), user2.ID)
	assert.NoError(t, err)
	assert.Equal(t, user2.ID, session2.UserID)
	assert.NotEqual(t, session1.ID, session2.ID)
	
	// Verify WebSocket broadcasts were sent to correct users
	receivedMessages := mockWS.GetReceivedMessages()
	assert.Len(t, receivedMessages, 2)
	
	// Check that each user got their message
	user1GotMessage := false
	user2GotMessage := false
	
	for _, msg := range receivedMessages {
		if msg.UserEmail == user1Email && msg.Message == "I'll help you with billing" {
			user1GotMessage = true
		}
		if msg.UserEmail == user2Email && msg.Message == "Let me assist with your technical question" {
			user2GotMessage = true
		}
	}
	
	assert.True(t, user1GotMessage)
	assert.True(t, user2GotMessage)
	
	// Stop service
	xmppService.Stop()
	
	mockWS.AssertExpectations(t)
}