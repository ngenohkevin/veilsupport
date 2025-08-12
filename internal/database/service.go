package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"veilsupport/internal/db"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool: pool,
	}
}

func (s *Service) Close() {
	s.pool.Close()
}

// User operations
func (s *Service) CreateUser(ctx context.Context, email, passwordHash, xmppJID string) (*db.User, error) {
	query := `
		INSERT INTO users (email, password_hash, xmpp_jid)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, xmpp_jid, created_at, updated_at`
	
	var user db.User
	err := s.pool.QueryRow(ctx, query, email, passwordHash, xmppJID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.XmppJid,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	query := `
		SELECT id, email, password_hash, xmpp_jid, created_at, updated_at 
		FROM users WHERE email = $1`
	
	var user db.User
	err := s.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.XmppJid,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.User{}, fmt.Errorf("user not found")
		}
		return db.User{}, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func (s *Service) GetUserByID(ctx context.Context, id string) (*db.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	query := `
		SELECT id, email, password_hash, xmpp_jid, created_at, updated_at 
		FROM users WHERE id = $1`
	
	var user db.User
	err = s.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.XmppJid,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &user, nil
}

func (s *Service) GetUserByXMPPJID(ctx context.Context, jid string) (*db.User, error) {
	query := `
		SELECT id, email, password_hash, xmpp_jid, created_at, updated_at 
		FROM users WHERE xmpp_jid = $1`
	
	var user db.User
	err := s.pool.QueryRow(ctx, query, jid).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.XmppJid,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by XMPP JID: %w", err)
	}
	return &user, nil
}

// Chat session operations
func (s *Service) CreateChatSession(ctx context.Context, userID uuid.UUID) (db.ChatSession, error) {
	query := `
		INSERT INTO chat_sessions (user_id)
		VALUES ($1)
		RETURNING id, user_id, status, created_at, updated_at`
	
	var session db.ChatSession
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&session.ID,
		&session.UserID,
		&session.Status,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return db.ChatSession{}, fmt.Errorf("failed to create chat session: %w", err)
	}
	return session, nil
}

func (s *Service) GetActiveSessionByUserID(ctx context.Context, userID uuid.UUID) (db.ChatSession, error) {
	query := `
		SELECT id, user_id, status, created_at, updated_at
		FROM chat_sessions 
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1`
	
	var session db.ChatSession
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&session.ID,
		&session.UserID,
		&session.Status,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.ChatSession{}, fmt.Errorf("no active session found")
		}
		return db.ChatSession{}, fmt.Errorf("failed to get active session: %w", err)
	}
	return session, nil
}

func (s *Service) GetOrCreateActiveSession(ctx context.Context, userID uuid.UUID) (db.ChatSession, error) {
	// Try to get existing active session
	session, err := s.GetActiveSessionByUserID(ctx, userID)
	if err == nil {
		return session, nil
	}
	
	// If no active session exists, create a new one
	return s.CreateChatSession(ctx, userID)
}

// Message operations
func (s *Service) SaveMessage(ctx context.Context, params db.SaveMessageParams) (db.Message, error) {
	query := `
		INSERT INTO messages (session_id, from_jid, to_jid, content, message_type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, session_id, from_jid, to_jid, content, message_type, sent_at, created_at`
	
	var message db.Message
	err := s.pool.QueryRow(ctx, query, params.SessionID, params.FromJid, params.ToJid, params.Content, params.MessageType).Scan(
		&message.ID,
		&message.SessionID,
		&message.FromJid,
		&message.ToJid,
		&message.Content,
		&message.MessageType,
		&message.SentAt,
		&message.CreatedAt,
	)
	if err != nil {
		return db.Message{}, fmt.Errorf("failed to save message: %w", err)
	}
	return message, nil
}

func (s *Service) GetMessagesBySession(ctx context.Context, sessionID uuid.UUID) ([]db.Message, error) {
	query := `
		SELECT id, session_id, from_jid, to_jid, content, message_type, sent_at, created_at
		FROM messages 
		WHERE session_id = $1 
		ORDER BY sent_at ASC`
	
	rows, err := s.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by session: %w", err)
	}
	defer rows.Close()
	
	var messages []db.Message
	for rows.Next() {
		var msg db.Message
		err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.FromJid,
			&msg.ToJid,
			&msg.Content,
			&msg.MessageType,
			&msg.SentAt,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over messages: %w", err)
	}
	
	return messages, nil
}

func (s *Service) GetRecentMessagesByUserID(ctx context.Context, params db.GetRecentMessagesByUserIDParams) ([]db.Message, error) {
	query := `
		SELECT m.id, m.session_id, m.from_jid, m.to_jid, m.content, m.message_type, m.sent_at, m.created_at
		FROM messages m
		JOIN chat_sessions cs ON m.session_id = cs.id
		WHERE cs.user_id = $1
		ORDER BY m.sent_at DESC
		LIMIT $2`
	
	rows, err := s.pool.Query(ctx, query, params.UserID, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}
	defer rows.Close()
	
	var messages []db.Message
	for rows.Next() {
		var msg db.Message
		err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.FromJid,
			&msg.ToJid,
			&msg.Content,
			&msg.MessageType,
			&msg.SentAt,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over messages: %w", err)
	}
	
	return messages, nil
}

func (s *Service) UpdateSessionStatus(ctx context.Context, params db.UpdateSessionStatusParams) error {
	query := `
		UPDATE chat_sessions 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2`
	
	_, err := s.pool.Exec(ctx, query, params.Status, params.ID)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}
	
	return nil
}