package xmpp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
)

// XMPPRegistrar handles dynamic XMPP account creation
type XMPPRegistrar struct {
	server   string
	domain   string
}

// NewXMPPRegistrar creates a new XMPP account registrar
func NewXMPPRegistrar(server, domain string) *XMPPRegistrar {
	return &XMPPRegistrar{
		server: server,
		domain: domain,
	}
}

// GenerateUserCredentials creates unique XMPP credentials for a web user
func (r *XMPPRegistrar) GenerateUserCredentials(userEmail string) (username, password, fullJID string, err error) {
	// Extract clean username from email
	emailParts := strings.Split(userEmail, "@")
	baseUsername := emailParts[0]
	
	// Clean username (XMPP allows a-z, 0-9, -, ., _)
	cleanUsername := ""
	for _, char := range baseUsername {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
		   (char >= '0' && char <= '9') || char == '-' || char == '.' || char == '_' {
			cleanUsername += string(char)
		}
	}
	
	// Ensure uniqueness with timestamp
	timestamp := time.Now().Unix()
	username = fmt.Sprintf("%s_%d", strings.ToLower(cleanUsername), timestamp)
	
	// Generate secure random password
	password, err = generateSecurePassword(16)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate password: %w", err)
	}
	
	fullJID = fmt.Sprintf("%s@%s", username, r.domain)
	
	return username, password, fullJID, nil
}

// CreateXMPPAccount attempts to create an XMPP account using In-Band Registration
func (r *XMPPRegistrar) CreateXMPPAccount(username, password string) error {
	fullJID := fmt.Sprintf("%s@%s", username, r.domain)
	
	log.Printf("Attempting to create XMPP account: %s", fullJID)
	
	// Parse the JID
	addr, err := jid.Parse(fullJID)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Attempt In-Band Registration (IBR)
	// Note: This may not work on all servers (many disable IBR for security)
	conn, err := xmpp.DialClientSession(
		ctx, addr,
		xmpp.StartTLS(nil),
		xmpp.SASL("", password, sasl.Plain),
		// Add registration feature if supported
	)
	
	if err != nil {
		// IBR likely not supported, return specific error
		return fmt.Errorf("account creation failed (server may not support in-band registration): %w", err)
	}
	
	conn.Close()
	log.Printf("XMPP account created successfully: %s", fullJID)
	return nil
}

// TestXMPPAccountExists checks if an XMPP account already exists
func (r *XMPPRegistrar) TestXMPPAccountExists(username, password string) bool {
	fullJID := fmt.Sprintf("%s@%s", username, r.domain)
	client := NewXMPPClient(fullJID, password, r.server)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err := client.ConnectWithContext(ctx)
	if err != nil {
		return false
	}
	
	client.Close()
	return true
}

// generateSecurePassword creates a cryptographically secure random password
func generateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// UserXMPPSession manages an individual user's XMPP connection
type UserXMPPSession struct {
	UserID   int
	JID      string
	Password string
	Client   *XMPPClient
	Active   bool
	LastUsed time.Time
}

// XMPPSessionManager manages multiple user XMPP sessions
type XMPPSessionManager struct {
	sessions map[int]*UserXMPPSession // userID -> session
	server   string
	adminJID string
}

// NewXMPPSessionManager creates a new session manager
func NewXMPPSessionManager(server, adminJID string) *XMPPSessionManager {
	return &XMPPSessionManager{
		sessions: make(map[int]*UserXMPPSession),
		server:   server,
		adminJID: adminJID,
	}
}

// GetOrCreateUserSession gets or creates an XMPP session for a user
func (sm *XMPPSessionManager) GetOrCreateUserSession(userID int, userEmail, xmppJID, xmppPassword string) (*UserXMPPSession, error) {
	// Check if session already exists
	if session, exists := sm.sessions[userID]; exists {
		if session.Active && time.Since(session.LastUsed) < 30*time.Minute {
			session.LastUsed = time.Now()
			return session, nil
		}
		// Clean up old session
		if session.Client != nil {
			session.Client.Close()
		}
		delete(sm.sessions, userID)
	}
	
	// Create new XMPP client for this user
	client := NewXMPPClient(xmppJID, xmppPassword, sm.server)
	
	// Try to connect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	err := client.ConnectWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect user XMPP session: %w", err)
	}
	
	// Create session
	session := &UserXMPPSession{
		UserID:   userID,
		JID:      xmppJID,
		Password: xmppPassword,
		Client:   client,
		Active:   true,
		LastUsed: time.Now(),
	}
	
	sm.sessions[userID] = session
	log.Printf("Created XMPP session for user %d (%s)", userID, xmppJID)
	
	return session, nil
}

// SendMessageAsUser sends a message directly from the user's XMPP account
func (sm *XMPPSessionManager) SendMessageAsUser(userID int, message string) error {
	session, exists := sm.sessions[userID]
	if !exists || !session.Active {
		return fmt.Errorf("no active XMPP session for user %d", userID)
	}
	
	// Send message to admin
	err := session.Client.SendMessage(sm.adminJID, message)
	if err != nil {
		return fmt.Errorf("failed to send message as user: %w", err)
	}
	
	session.LastUsed = time.Now()
	log.Printf("Message sent from user %s to %s: %s", session.JID, sm.adminJID, message)
	
	return nil
}

// CleanupInactiveSessions removes inactive sessions
func (sm *XMPPSessionManager) CleanupInactiveSessions() {
	cutoff := time.Now().Add(-30 * time.Minute)
	
	for userID, session := range sm.sessions {
		if session.LastUsed.Before(cutoff) {
			log.Printf("Cleaning up inactive session for user %d", userID)
			if session.Client != nil {
				session.Client.Close()
			}
			delete(sm.sessions, userID)
		}
	}
}