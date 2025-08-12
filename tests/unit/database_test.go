package unit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"veilsupport/internal/database"
	"veilsupport/tests/testhelpers"
)

func TestDatabaseService_CreateUser(t *testing.T) {
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()
	email := "test@example.com"
	passwordHash := "hashedpassword123"
	xmppJID := "test@veilsupport.local"

	// Test successful user creation
	user, err := service.CreateUser(ctx, email, passwordHash, xmppJID)
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, email, user.Email)
	assert.Equal(t, passwordHash, user.PasswordHash)
	assert.Equal(t, xmppJID, user.XmppJid)
	assert.NotEmpty(t, user.ID)
	assert.False(t, user.CreatedAt.Time.IsZero())
}

func TestDatabaseService_CreateUser_DuplicateEmail(t *testing.T) {
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()
	email := "duplicate@example.com"
	passwordHash := "hashedpassword123"
	xmppJID1 := "test1@veilsupport.local"
	xmppJID2 := "test2@veilsupport.local"

	// Create first user
	_, err := service.CreateUser(ctx, email, passwordHash, xmppJID1)
	require.NoError(t, err)

	// Try to create user with same email
	_, err = service.CreateUser(ctx, email, passwordHash, xmppJID2)
	assert.Error(t, err)
}

func TestDatabaseService_GetUserByEmail(t *testing.T) {
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()
	email := "getbyemail@example.com"
	passwordHash := "hashedpassword123"
	xmppJID := "getbyemail@veilsupport.local"

	// Create user first
	createdUser, err := service.CreateUser(ctx, email, passwordHash, xmppJID)
	require.NoError(t, err)

	// Get user by email
	retrievedUser, err := service.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)

	assert.Equal(t, createdUser.ID, retrievedUser.ID)
	assert.Equal(t, createdUser.Email, retrievedUser.Email)
	assert.Equal(t, createdUser.PasswordHash, retrievedUser.PasswordHash)
	assert.Equal(t, createdUser.XmppJid, retrievedUser.XmppJid)
}

func TestDatabaseService_GetUserByEmail_NotFound(t *testing.T) {
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()

	// Try to get non-existent user
	_, err := service.GetUserByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)
}

func TestDatabaseService_ChatSession_Lifecycle(t *testing.T) {
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()

	// Create a test user first
	user, err := service.CreateUser(ctx, "sessiontest@example.com", "hash123", "sessiontest@veilsupport.local")
	require.NoError(t, err)

	userIDStr := user.ID.String()

	// Test creating chat session
	session, err := service.CreateChatSession(ctx, userIDStr)
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.Equal(t, user.ID, session.UserID)
	assert.Equal(t, "active", session.Status)
	assert.NotEmpty(t, session.ID)

	// Test getting active session
	activeSession, err := service.GetActiveSessionByUserID(ctx, userIDStr)
	require.NoError(t, err)
	assert.Equal(t, session.ID, activeSession.ID)

	// Test GetOrCreateActiveSession with existing session
	foundSession, err := service.GetOrCreateActiveSession(ctx, userIDStr)
	require.NoError(t, err)
	assert.Equal(t, session.ID, foundSession.ID)
}

func TestDatabaseService_Messages(t *testing.T) {
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()

	// Create a test user and session
	user, err := service.CreateUser(ctx, "messagetest@example.com", "hash123", "messagetest@veilsupport.local")
	require.NoError(t, err)

	session, err := service.CreateChatSession(ctx, user.ID.String())
	require.NoError(t, err)

	// Test saving a message
	sessionIDStr := session.ID.String()
	fromJID := user.XmppJid
	toJID := "admin@veilsupport.local"
	content := "Hello, I need help!"
	messageType := "user"

	message, err := service.SaveMessage(ctx, sessionIDStr, fromJID, toJID, content, messageType)
	require.NoError(t, err)
	require.NotNil(t, message)

	assert.Equal(t, session.ID, message.SessionID)
	assert.Equal(t, fromJID, message.FromJid)
	assert.Equal(t, toJID, message.ToJid)
	assert.Equal(t, content, message.Content)
	assert.Equal(t, messageType, message.MessageType)

	// Test getting messages by session
	messages, err := service.GetMessagesBySession(ctx, sessionIDStr)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, message.ID, messages[0].ID)

	// Test getting recent messages by user ID
	recentMessages, err := service.GetRecentMessagesByUserID(ctx, user.ID.String(), 10)
	require.NoError(t, err)
	require.Len(t, recentMessages, 1)
	assert.Equal(t, message.ID, recentMessages[0].ID)
}

func TestDatabaseService_GetOrCreateActiveSession_NewSession(t *testing.T) {
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()

	// Create a test user
	user, err := service.CreateUser(ctx, "newsession@example.com", "hash123", "newsession@veilsupport.local")
	require.NoError(t, err)

	// Test GetOrCreateActiveSession when no session exists
	session, err := service.GetOrCreateActiveSession(ctx, user.ID.String())
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.Equal(t, user.ID, session.UserID)
	assert.Equal(t, "active", session.Status)
}

func TestDatabaseService_InvalidUUID(t *testing.T) {
	pool := testhelpers.SetupTestDB(t)
	service := database.NewService(pool)

	ctx := context.Background()

	// Test invalid UUID for GetUserByID
	_, err := service.GetUserByID(ctx, "invalid-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")

	// Test invalid UUID for CreateChatSession
	_, err = service.CreateChatSession(ctx, "invalid-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")

	// Test invalid UUID for SaveMessage
	_, err = service.SaveMessage(ctx, "invalid-uuid", "from@test.com", "to@test.com", "content", "user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid session ID")
}