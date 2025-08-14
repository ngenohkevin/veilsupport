package tests

import (
	"context"
	"os"
	"testing"

	"github.com/ngenohkevin/veilsupport/internal/db"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *db.DB {
	// Use test database URL or fallback
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://veiluser:veilpass@localhost:5433/veilsupport_test?sslmode=disable"
	}

	database, err := db.New(dbURL)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}

	// Clean up tables before each test
	cleanupTestDB(t, database)
	
	// Run migrations
	runTestMigrations(t, database)

	return database
}

func cleanupTestDB(t *testing.T, database *db.DB) {
	// Drop tables if they exist
	_, err := database.GetConn().Exec(context.Background(), "DROP TABLE IF EXISTS messages CASCADE")
	assert.NoError(t, err)
	_, err = database.GetConn().Exec(context.Background(), "DROP TABLE IF EXISTS users CASCADE")
	assert.NoError(t, err)
}

func runTestMigrations(t *testing.T, database *db.DB) {
	// Create users table
	_, err := database.GetConn().Exec(context.Background(), `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			xmpp_jid VARCHAR(255) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	assert.NoError(t, err)

	// Create messages table
	_, err = database.GetConn().Exec(context.Background(), `
		CREATE TABLE messages (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			content TEXT NOT NULL,
			sender_type VARCHAR(20) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	assert.NoError(t, err)

	// Create index
	_, err = database.GetConn().Exec(context.Background(), `
		CREATE INDEX idx_messages_user_id ON messages(user_id)
	`)
	assert.NoError(t, err)
}

func createTestUser(t *testing.T, database *db.DB) *db.User {
	user, err := database.CreateUser("test@example.com", "hashedpass")
	assert.NoError(t, err)
	return user
}

func TestUserCreation(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	user, err := database.CreateUser("test@example.com", "hashedpass")
	assert.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.NotEmpty(t, user.XmppJID)
	assert.Contains(t, user.XmppJID, "user_")
}

func TestMessageStorage(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	user := createTestUser(t, database)

	msg, err := database.SaveMessage(user.ID, "Hello", "user")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", msg.Content)
	assert.Equal(t, "user", msg.SenderType)
	assert.Equal(t, user.ID, msg.UserID)
}

func TestGetUserMessages(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	user := createTestUser(t, database)

	_, err := database.SaveMessage(user.ID, "Message 1", "user")
	assert.NoError(t, err)
	_, err = database.SaveMessage(user.ID, "Message 2", "admin")
	assert.NoError(t, err)

	messages, err := database.GetUserMessages(user.ID)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "Message 1", messages[0].Content)
	assert.Equal(t, "Message 2", messages[1].Content)
}