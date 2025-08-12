package unit

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"veilsupport/internal/chat"
	"veilsupport/internal/db"
)

type MockDatabaseService struct {
	mock.Mock
}

func (m *MockDatabaseService) SaveMessage(ctx context.Context, params db.SaveMessageParams) (db.Message, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(db.Message), args.Error(1)
}

func (m *MockDatabaseService) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(db.User), args.Error(1)
}

func (m *MockDatabaseService) GetActiveSessionByUserID(ctx context.Context, userID uuid.UUID) (db.ChatSession, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(db.ChatSession), args.Error(1)
}

func (m *MockDatabaseService) CreateChatSession(ctx context.Context, userID uuid.UUID) (db.ChatSession, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(db.ChatSession), args.Error(1)
}

func (m *MockDatabaseService) GetRecentMessagesByUserID(ctx context.Context, params db.GetRecentMessagesByUserIDParams) ([]db.Message, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]db.Message), args.Error(1)
}

func (m *MockDatabaseService) GetMessagesBySession(ctx context.Context, sessionID uuid.UUID) ([]db.Message, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]db.Message), args.Error(1)
}

func (m *MockDatabaseService) UpdateSessionStatus(ctx context.Context, params db.UpdateSessionStatusParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

type MockChatWSManager struct {
	mock.Mock
}

func (m *MockChatWSManager) SendToUser(userEmail, message string) error {
	args := m.Called(userEmail, message)
	return args.Error(0)
}

func TestChatService_SaveMessage(t *testing.T) {
	mockDB := &MockDatabaseService{}
	mockWS := &MockChatWSManager{}
	service := chat.NewService(mockDB, mockWS)

	ctx := context.Background()
	sessionID := uuid.New().String()
	fromJID := "user123@test.local"
	toJID := "admin@test.local"
	content := "Hello, I need help!"
	messageType := "user"

	// Setup expectations
	now := time.Now()
	expectedMessage := db.Message{
		ID:          uuid.New(),
		SessionID:   uuid.MustParse(sessionID),
		FromJid:     fromJID,
		ToJid:       toJID,
		Content:     content,
		MessageType: messageType,
		SentAt:      sql.NullTime{Time: now, Valid: true},
		CreatedAt:   sql.NullTime{Time: now, Valid: true},
	}

	mockDB.On("SaveMessage", ctx, mock.MatchedBy(func(params db.SaveMessageParams) bool {
		return params.SessionID.String() == sessionID &&
			params.FromJid == fromJID &&
			params.ToJid == toJID &&
			params.Content == content &&
			params.MessageType == messageType
	})).Return(expectedMessage, nil)

	// Test
	err := service.SaveMessage(ctx, sessionID, fromJID, toJID, content, messageType)

	// Assert
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestChatService_GetOrCreateSession_ExistingSession(t *testing.T) {
	mockDB := &MockDatabaseService{}
	mockWS := &MockChatWSManager{}
	service := chat.NewService(mockDB, mockWS)

	ctx := context.Background()
	userEmail := "test@example.com"
	userID := uuid.New()
	sessionID := uuid.New()

	// Setup expectations
	now := time.Now()
	expectedUser := db.User{
		ID:           userID,
		Email:        userEmail,
		PasswordHash: "hashedpassword",
		XmppJid:      "user_test_example_com@test.local",
		CreatedAt:    sql.NullTime{Time: now, Valid: true},
		UpdatedAt:    sql.NullTime{Time: now, Valid: true},
	}

	expectedSession := db.ChatSession{
		ID:        sessionID,
		UserID:    userID,
		Status:    "active",
		CreatedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
	}

	mockDB.On("GetUserByEmail", ctx, userEmail).Return(expectedUser, nil)
	mockDB.On("GetActiveSessionByUserID", ctx, userID).Return(expectedSession, nil)

	// Test
	result, err := service.GetOrCreateSession(ctx, userEmail)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, sessionID.String(), result)
	mockDB.AssertExpectations(t)
}

func TestChatService_GetOrCreateSession_NewSession(t *testing.T) {
	mockDB := &MockDatabaseService{}
	mockWS := &MockChatWSManager{}
	service := chat.NewService(mockDB, mockWS)

	ctx := context.Background()
	userEmail := "test@example.com"
	userID := uuid.New()
	sessionID := uuid.New()

	// Setup expectations
	now := time.Now()
	expectedUser := db.User{
		ID:           userID,
		Email:        userEmail,
		PasswordHash: "hashedpassword",
		XmppJid:      "user_test_example_com@test.local",
		CreatedAt:    sql.NullTime{Time: now, Valid: true},
		UpdatedAt:    sql.NullTime{Time: now, Valid: true},
	}

	expectedSession := db.ChatSession{
		ID:        sessionID,
		UserID:    userID,
		Status:    "active",
		CreatedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
	}

	mockDB.On("GetUserByEmail", ctx, userEmail).Return(expectedUser, nil)
	mockDB.On("GetActiveSessionByUserID", ctx, userID).Return(db.ChatSession{}, assert.AnError) // No active session
	mockDB.On("CreateChatSession", ctx, userID).Return(expectedSession, nil)

	// Test
	result, err := service.GetOrCreateSession(ctx, userEmail)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, sessionID.String(), result)
	mockDB.AssertExpectations(t)
}

func TestChatService_BroadcastToWebSocket(t *testing.T) {
	mockDB := &MockDatabaseService{}
	mockWS := &MockChatWSManager{}
	service := chat.NewService(mockDB, mockWS)

	userEmail := "test@example.com"
	message := "Admin response"

	// Setup expectations
	mockWS.On("SendToUser", userEmail, message).Return(nil)

	// Test
	err := service.BroadcastToWebSocket(userEmail, message)

	// Assert
	assert.NoError(t, err)
	mockWS.AssertExpectations(t)
}

func TestChatService_BroadcastToWebSocket_NoWSManager(t *testing.T) {
	mockDB := &MockDatabaseService{}
	service := chat.NewService(mockDB, nil) // No WebSocket manager

	userEmail := "test@example.com"
	message := "Admin response"

	// Test
	err := service.BroadcastToWebSocket(userEmail, message)

	// Assert - should not error when WS manager is nil
	assert.NoError(t, err)
}

func TestChatService_GetMessageHistory(t *testing.T) {
	mockDB := &MockDatabaseService{}
	mockWS := &MockChatWSManager{}
	service := chat.NewService(mockDB, mockWS)

	ctx := context.Background()
	userEmail := "test@example.com"
	userID := uuid.New()
	limit := int32(50)

	// Setup expectations
	now := time.Now()
	expectedUser := db.User{
		ID:           userID,
		Email:        userEmail,
		PasswordHash: "hashedpassword",
		XmppJid:      "user_test_example_com@test.local",
		CreatedAt:    sql.NullTime{Time: now, Valid: true},
		UpdatedAt:    sql.NullTime{Time: now, Valid: true},
	}
	expectedMessages := []db.Message{
		{
			ID:          uuid.New(),
			SessionID:   uuid.New(),
			FromJid:     "user_test_example_com@test.local",
			ToJid:       "admin@test.local",
			Content:     "Hello, I need help!",
			MessageType: "user",
			SentAt:      sql.NullTime{Time: now, Valid: true},
			CreatedAt:   sql.NullTime{Time: now, Valid: true},
		},
		{
			ID:          uuid.New(),
			SessionID:   uuid.New(),
			FromJid:     "admin@test.local",
			ToJid:       "user_test_example_com@test.local",
			Content:     "How can I assist you?",
			MessageType: "admin",
			SentAt:      sql.NullTime{Time: now, Valid: true},
			CreatedAt:   sql.NullTime{Time: now, Valid: true},
		},
	}

	mockDB.On("GetUserByEmail", ctx, userEmail).Return(expectedUser, nil)
	mockDB.On("GetRecentMessagesByUserID", ctx, db.GetRecentMessagesByUserIDParams{
		UserID: userID,
		Limit:  limit,
	}).Return(expectedMessages, nil)

	// Test
	result, err := service.GetMessageHistory(ctx, userEmail, limit)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedMessages, result)
	assert.Len(t, result, 2)
	mockDB.AssertExpectations(t)
}

func TestChatService_CloseSession(t *testing.T) {
	mockDB := &MockDatabaseService{}
	mockWS := &MockChatWSManager{}
	service := chat.NewService(mockDB, mockWS)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// Setup expectations
	mockDB.On("UpdateSessionStatus", ctx, db.UpdateSessionStatusParams{
		Status: "closed",
		ID:     uuid.MustParse(sessionID),
	}).Return(nil)

	// Test
	err := service.CloseSession(ctx, sessionID)

	// Assert
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}