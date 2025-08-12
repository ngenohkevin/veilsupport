package chat

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"veilsupport/internal/db"
)

type DatabaseService interface {
	SaveMessage(ctx context.Context, params db.SaveMessageParams) (db.Message, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetActiveSessionByUserID(ctx context.Context, userID uuid.UUID) (db.ChatSession, error)
	CreateChatSession(ctx context.Context, userID uuid.UUID) (db.ChatSession, error)
	GetRecentMessagesByUserID(ctx context.Context, params db.GetRecentMessagesByUserIDParams) ([]db.Message, error)
	GetMessagesBySession(ctx context.Context, sessionID uuid.UUID) ([]db.Message, error)
	UpdateSessionStatus(ctx context.Context, params db.UpdateSessionStatusParams) error
}

type Service struct {
	dbService DatabaseService
	wsManager WSManager
}

type WSManager interface {
	SendToUser(userEmail, message string) error
}

func NewService(dbService DatabaseService, wsManager WSManager) *Service {
	return &Service{
		dbService: dbService,
		wsManager: wsManager,
	}
}

func (s *Service) SaveMessage(ctx context.Context, sessionID, fromJID, toJID, content, messageType string) error {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	params := db.SaveMessageParams{
		SessionID:   sessionUUID,
		FromJid:     fromJID,
		ToJid:       toJID,
		Content:     content,
		MessageType: messageType,
	}

	_, err = s.dbService.SaveMessage(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	log.Printf("Saved message from %s to %s in session %s", fromJID, toJID, sessionID)
	return nil
}

func (s *Service) GetOrCreateSession(ctx context.Context, userEmail string) (string, error) {
	// First, try to get the user by email
	user, err := s.dbService.GetUserByEmail(ctx, userEmail)
	if err != nil {
		// If user doesn't exist, this would be where we handle user creation
		// For now, return an error since users should be registered first
		return "", fmt.Errorf("user not found: %s", userEmail)
	}

	// Try to get an active session for this user
	session, err := s.dbService.GetActiveSessionByUserID(ctx, user.ID)
	if err != nil {
		// No active session found, create a new one
		newSession, createErr := s.dbService.CreateChatSession(ctx, user.ID)
		if createErr != nil {
			return "", fmt.Errorf("failed to create new session: %w", createErr)
		}
		return newSession.ID.String(), nil
	}

	return session.ID.String(), nil
}

func (s *Service) BroadcastToWebSocket(userEmail, message string) error {
	if s.wsManager == nil {
		// WebSocket manager not available (e.g., in tests)
		log.Printf("WebSocket manager not available, skipping broadcast to %s", userEmail)
		return nil
	}

	err := s.wsManager.SendToUser(userEmail, message)
	if err != nil {
		return fmt.Errorf("failed to send WebSocket message to %s: %w", userEmail, err)
	}

	log.Printf("Broadcasted message to %s via WebSocket", userEmail)
	return nil
}

func (s *Service) GetMessageHistory(ctx context.Context, userEmail string, limit int32) ([]db.Message, error) {
	// First, get the user by email
	user, err := s.dbService.GetUserByEmail(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s", userEmail)
	}

	// Get recent messages for this user
	messages, err := s.dbService.GetRecentMessagesByUserID(ctx, db.GetRecentMessagesByUserIDParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	return messages, nil
}

func (s *Service) GetSessionHistory(ctx context.Context, sessionID string) ([]db.Message, error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	messages, err := s.dbService.GetMessagesBySession(ctx, sessionUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session history: %w", err)
	}

	return messages, nil
}

func (s *Service) CloseSession(ctx context.Context, sessionID string) error {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	err = s.dbService.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		Status: "closed",
		ID:     sessionUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	log.Printf("Closed session %s", sessionID)
	return nil
}