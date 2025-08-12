package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"veilsupport/internal/database"
	"veilsupport/internal/db"
	"veilsupport/tests/testhelpers"
)

func TestDatabaseIntegration_CompleteUserJourney(t *testing.T) {
	// Setup test database
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()

	// Step 1: Create a user
	email := "journey@example.com"
	passwordHash := "hashedpassword123"
	xmppJID := "journey@veilsupport.local"

	user, err := service.CreateUser(ctx, email, passwordHash, xmppJID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Step 2: Create a chat session for the user
	session, err := service.GetOrCreateActiveSession(ctx, user.ID.String())
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, user.ID, session.UserID)
	assert.Equal(t, "active", session.Status)

	// Step 3: User sends a message
	userMessage, err := service.SaveMessage(ctx, session.ID.String(), user.XmppJid, "admin@veilsupport.local", "Hello, I need help with my order!", "user")
	require.NoError(t, err)
	assert.Equal(t, "user", userMessage.MessageType)
	assert.Equal(t, "Hello, I need help with my order!", userMessage.Content)

	// Step 4: Admin replies
	adminMessage, err := service.SaveMessage(ctx, session.ID.String(), "admin@veilsupport.local", user.XmppJid, "Hello! I'd be happy to help you with your order. What's the order number?", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", adminMessage.MessageType)

	// Step 5: User sends another message
	_, err = service.SaveMessage(ctx, session.ID.String(), user.XmppJid, "admin@veilsupport.local", "My order number is #12345", "user")
	require.NoError(t, err)

	// Step 6: Get all messages for the session
	messages, err := service.GetMessagesBySession(ctx, session.ID.String())
	require.NoError(t, err)
	require.Len(t, messages, 3)

	// Verify message order (should be chronological)
	assert.Equal(t, "Hello, I need help with my order!", messages[0].Content)
	assert.Equal(t, "user", messages[0].MessageType)

	assert.Equal(t, "Hello! I'd be happy to help you with your order. What's the order number?", messages[1].Content)
	assert.Equal(t, "admin", messages[1].MessageType)

	assert.Equal(t, "My order number is #12345", messages[2].Content)
	assert.Equal(t, "user", messages[2].MessageType)

	// Step 7: Get recent messages for user
	recentMessages, err := service.GetRecentMessagesByUserID(ctx, user.ID.String(), 10)
	require.NoError(t, err)
	require.Len(t, recentMessages, 3)

	// Recent messages should be in reverse chronological order
	assert.Equal(t, "My order number is #12345", recentMessages[0].Content)
	assert.Equal(t, "Hello! I'd be happy to help you with your order. What's the order number?", recentMessages[1].Content)
	assert.Equal(t, "Hello, I need help with my order!", recentMessages[2].Content)

	// Step 8: Verify user can be retrieved by different methods
	retrievedByEmail, err := service.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedByEmail.ID)

	retrievedByID, err := service.GetUserByID(ctx, user.ID.String())
	require.NoError(t, err)
	assert.Equal(t, user.Email, retrievedByID.Email)

	retrievedByJID, err := service.GetUserByXMPPJID(ctx, xmppJID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedByJID.ID)

	// Step 9: Test creating another session should return the same active session
	sameSession, err := service.GetOrCreateActiveSession(ctx, user.ID.String())
	require.NoError(t, err)
	assert.Equal(t, session.ID, sameSession.ID)
}

func TestDatabaseIntegration_MultipleUsersAndSessions(t *testing.T) {
	// Setup test database
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()

	// Create multiple users
	users := make([]*db.User, 3)
	for i := 0; i < 3; i++ {
		email := fmt.Sprintf("user%d@example.com", i+1)
		passwordHash := "hashedpassword123"
		xmppJID := fmt.Sprintf("user%d@veilsupport.local", i+1)

		user, err := service.CreateUser(ctx, email, passwordHash, xmppJID)
		require.NoError(t, err)
		users[i] = user
	}

	// Create sessions and messages for each user
	for i, user := range users {
		session, err := service.GetOrCreateActiveSession(ctx, user.ID.String())
		require.NoError(t, err)

		// Each user sends a message
		message := fmt.Sprintf("Message from user %d", i+1)
		_, err = service.SaveMessage(ctx, session.ID.String(), user.XmppJid, "admin@veilsupport.local", message, "user")
		require.NoError(t, err)

		// Admin replies to each user
		reply := fmt.Sprintf("Reply to user %d", i+1)
		_, err = service.SaveMessage(ctx, session.ID.String(), "admin@veilsupport.local", user.XmppJid, reply, "admin")
		require.NoError(t, err)
	}

	// Verify each user has their own separate conversation
	for i, user := range users {
		messages, err := service.GetRecentMessagesByUserID(ctx, user.ID.String(), 10)
		require.NoError(t, err)
		require.Len(t, messages, 2)

		// Check that user only sees their own messages
		expectedUserMessage := fmt.Sprintf("Message from user %d", i+1)
		expectedAdminReply := fmt.Sprintf("Reply to user %d", i+1)

		// Messages are in reverse chronological order
		assert.Contains(t, messages[0].Content, fmt.Sprintf("user %d", i+1))
		assert.Contains(t, messages[1].Content, fmt.Sprintf("user %d", i+1))

		// Verify content matches
		messageContents := []string{messages[0].Content, messages[1].Content}
		assert.Contains(t, messageContents, expectedUserMessage)
		assert.Contains(t, messageContents, expectedAdminReply)
	}
}