package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	conn *pgx.Conn
}

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Don't include in JSON responses
	XmppJID      string    `json:"xmpp_jid"`
	CreatedAt    time.Time `json:"created_at"`
}

type Message struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Content    string    `json:"content"`
	SenderType string    `json:"sender_type"`
	CreatedAt  time.Time `json:"created_at"`
}

func New(dsn string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &DB{conn: conn}, nil
}

func (d *DB) Close() error {
	return d.conn.Close(context.Background())
}

func (d *DB) GetConn() *pgx.Conn {
	return d.conn
}

func generateJID(email string) string {
	return fmt.Sprintf("user_%d@domain.com", time.Now().Unix())
}

func (d *DB) CreateUser(email, passwordHash string) (*User, error) {
	xmppJID := generateJID(email)
	var user User
	
	err := d.conn.QueryRow(context.Background(),
		`INSERT INTO users (email, password_hash, xmpp_jid) 
         VALUES ($1, $2, $3) RETURNING id, email, xmpp_jid, created_at`,
		email, passwordHash, xmppJID).Scan(&user.ID, &user.Email, &user.XmppJID, &user.CreatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return &user, nil
}

func (d *DB) GetUserByEmail(email string) (*User, error) {
	var user User
	
	err := d.conn.QueryRow(context.Background(),
		`SELECT id, email, password_hash, xmpp_jid, created_at FROM users WHERE email = $1`,
		email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.XmppJID, &user.CreatedAt)
	
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	
	return &user, nil
}

func (d *DB) GetUserByID(id int) (*User, error) {
	var user User
	
	err := d.conn.QueryRow(context.Background(),
		`SELECT id, email, xmpp_jid, created_at FROM users WHERE id = $1`,
		id).Scan(&user.ID, &user.Email, &user.XmppJID, &user.CreatedAt)
	
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	
	return &user, nil
}

func (d *DB) GetUserByJID(jid string) (*User, error) {
	var user User
	
	err := d.conn.QueryRow(context.Background(),
		`SELECT id, email, xmpp_jid, created_at FROM users WHERE xmpp_jid = $1`,
		jid).Scan(&user.ID, &user.Email, &user.XmppJID, &user.CreatedAt)
	
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by JID: %w", err)
	}
	
	return &user, nil
}

func (d *DB) SaveMessage(userID int, content, senderType string) (*Message, error) {
	var msg Message
	
	err := d.conn.QueryRow(context.Background(),
		`INSERT INTO messages (user_id, content, sender_type) 
         VALUES ($1, $2, $3) RETURNING id, user_id, content, sender_type, created_at`,
		userID, content, senderType).Scan(&msg.ID, &msg.UserID, &msg.Content, &msg.SenderType, &msg.CreatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}
	
	return &msg, nil
}

func (d *DB) GetUserMessages(userID int) ([]Message, error) {
	rows, err := d.conn.Query(context.Background(),
		`SELECT id, user_id, content, sender_type, created_at FROM messages 
         WHERE user_id = $1 ORDER BY created_at`, userID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get user messages: %w", err)
	}
	defer rows.Close()
	
	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.Content, &msg.SenderType, &msg.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}
	
	return messages, nil
}