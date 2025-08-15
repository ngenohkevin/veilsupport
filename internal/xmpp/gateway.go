package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// GatewayClient acts as a bridge between web users and XMPP
// It uses a single bot account to represent ALL web users
type GatewayClient struct {
	botJID    string           // The bot's JID (e.g., veilbot@xmpp.jp)
	password  string           // Bot's password
	server    string           // XMPP server
	adminJIDs []string         // Admin JIDs to receive messages
	session   *xmpp.Session    // XMPP session
	connected bool             // Connection status
	userMap   map[int]UserInfo // Map of userID to user info
	mu        sync.RWMutex     // Mutex for thread safety
}

// UserInfo represents a web user in the XMPP context
type UserInfo struct {
	UserID      int
	Email       string
	DisplayName string
	ResourceID  string // e.g., "user_123_john"
	IsOnline    bool
	LastSeen    time.Time
}

// GatewayMessage represents a message through the gateway
type GatewayMessage struct {
	UserID      int
	UserEmail   string
	DisplayName string
	Body        string
	Attachments []string
	FromAdmin   bool
	Timestamp   time.Time
}

// NewGatewayClient creates a new XMPP gateway client
func NewGatewayClient(botJID, password, server string, adminJIDs []string) *GatewayClient {
	return &GatewayClient{
		botJID:    botJID,
		password:  password,
		server:    server,
		adminJIDs: adminJIDs,
		userMap:   make(map[int]UserInfo),
	}
}

// Connect establishes connection to XMPP server as the bot
func (g *GatewayClient) Connect(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.connected && g.session != nil {
		return nil
	}

	// Parse bot JID
	addr, err := jid.Parse(g.botJID)
	if err != nil {
		return fmt.Errorf("invalid bot JID: %w", err)
	}

	log.Printf("Gateway: Connecting to %s as bot %s", g.server, g.botJID)

	// TLS config
	tlsConfig := &tls.Config{
		ServerName:         addr.Domain().String(),
		InsecureSkipVerify: true,
	}

	// Connect to XMPP server
	session, err := xmpp.DialClientSession(
		ctx, addr,
		xmpp.BindResource(),
		xmpp.StartTLS(tlsConfig),
		xmpp.SASL("", g.password, sasl.Plain),
	)
	if err != nil {
		return fmt.Errorf("failed to create gateway session: %w", err)
	}

	// Send presence
	err = session.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		session.Close()
		return fmt.Errorf("failed to send presence: %w", err)
	}

	g.session = session
	g.connected = true

	log.Printf("Gateway: Successfully connected as %s", g.botJID)
	return nil
}

// RegisterUser registers a web user with the gateway
func (g *GatewayClient) RegisterUser(userID int, email, displayName string) string {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Generate resource ID for this user
	resourceID := g.generateResourceID(userID, displayName)

	g.userMap[userID] = UserInfo{
		UserID:      userID,
		Email:       email,
		DisplayName: displayName,
		ResourceID:  resourceID,
		IsOnline:    true,
		LastSeen:    time.Now(),
	}

	log.Printf("Gateway: Registered user %s (%s) as %s", displayName, email, resourceID)
	return resourceID
}

// SendUserMessage sends a message from a web user to admin
func (g *GatewayClient) SendUserMessage(userID int, messageBody string, attachments []string) error {
	g.mu.RLock()
	user, exists := g.userMap[userID]
	g.mu.RUnlock()

	if !exists {
		return fmt.Errorf("user %d not registered with gateway", userID)
	}

	if !g.connected || g.session == nil {
		return errors.New("gateway not connected to XMPP server")
	}

	// Send to each admin
	for _, adminJID := range g.adminJIDs {
		err := g.sendMessageAsUser(user, adminJID, messageBody, attachments)
		if err != nil {
			log.Printf("Gateway: Failed to send to admin %s: %v", adminJID, err)
		}
	}

	return nil
}

// sendMessageAsUser sends a message that appears to come from a specific user
func (g *GatewayClient) sendMessageAsUser(user UserInfo, toJID, body string, attachments []string) error {
	// Parse recipient JID
	recipientJID, err := jid.Parse(toJID)
	if err != nil {
		return fmt.Errorf("invalid recipient JID: %w", err)
	}

	// Create message with enhanced user identification
	// Format that makes it easy to identify and reply to users
	formattedBody := fmt.Sprintf("👤 %s <%s>\n📧 User ID: %d\n\n💬 %s", 
		user.DisplayName, user.Email, user.UserID, body)

	// Add attachment info if present
	if len(attachments) > 0 {
		formattedBody += fmt.Sprintf("\n\n📎 Attachments: %d file(s)", len(attachments))
		for _, url := range attachments {
			formattedBody += fmt.Sprintf("\n• %s", url)
		}
	}

	// Create message from the bot account (XMPP doesn't allow spoofing "from" field)
	// Instead, we'll use the message subject and body to identify users clearly
	msg := stanza.Message{
		To:   recipientJID,
		Type: stanza.ChatMessage,
		ID:   fmt.Sprintf("msg_%d_%d", user.UserID, time.Now().Unix()),
	}

	// Send message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create message body element (back to working approach)
	bodyStart := xml.StartElement{Name: xml.Name{Local: "body"}}
	bodyContent := xmlstream.Wrap(
		xmlstream.Token(xml.CharData(formattedBody)),
		bodyStart,
	)
	
	// Wrap the message with body content
	messageWithBody := msg.Wrap(bodyContent)
	
	// Send message
	err = g.session.Send(ctx, messageWithBody)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("Gateway: Message from %s sent to %s", user.DisplayName, toJID)
	return nil
}

// HandleAdminReply processes replies from admin to web users
func (g *GatewayClient) HandleAdminReply(_, body string) (*GatewayMessage, error) {
	// Extract user ID from the message thread or context
	userID := g.extractUserIDFromMessage(body)
	if userID == 0 {
		return nil, fmt.Errorf("could not determine target user from admin message")
	}

	g.mu.RLock()
	user, exists := g.userMap[userID]
	g.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("user %d not found", userID)
	}

	// Create gateway message for routing to web user
	gwMsg := &GatewayMessage{
		UserID:      user.UserID,
		UserEmail:   user.Email,
		DisplayName: user.DisplayName,
		Body:        body,
		FromAdmin:   true,
		Timestamp:   time.Now(),
	}

	log.Printf("Gateway: Admin reply routed to user %s", user.DisplayName)
	return gwMsg, nil
}

// SetUserOnline updates user's online status
func (g *GatewayClient) SetUserOnline(userID int, online bool) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	user, exists := g.userMap[userID]
	if !exists {
		return fmt.Errorf("user %d not found", userID)
	}

	user.IsOnline = online
	user.LastSeen = time.Now()
	g.userMap[userID] = user

	// Send presence update to admins
	if g.connected && g.session != nil {
		presenceType := stanza.AvailablePresence
		if !online {
			presenceType = stanza.UnavailablePresence
		}

		for _, adminJID := range g.adminJIDs {
			if err := g.sendPresenceUpdate(user, adminJID, presenceType); err != nil {
				log.Printf("Gateway: Failed to send presence to %s: %v", adminJID, err)
			}
		}
	}

	return nil
}

// sendPresenceUpdate sends presence information about a user
func (g *GatewayClient) sendPresenceUpdate(user UserInfo, toJID string, presenceType stanza.PresenceType) error {
	recipientJID, err := jid.Parse(toJID)
	if err != nil {
		return err
	}

	// Create presence with status
	pres := stanza.Presence{
		To:   recipientJID,
		Type: presenceType,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create status element (using the working pattern)
	statusStart := xml.StartElement{Name: xml.Name{Local: "status"}}
	statusContent := xmlstream.Wrap(
		xmlstream.Token(xml.CharData(fmt.Sprintf("%s (%s)", user.DisplayName, user.Email))),
		statusStart,
	)
	
	// Wrap the presence with status content
	presenceWithStatus := pres.Wrap(statusContent)
	
	return g.session.Send(ctx, presenceWithStatus)
}

// generateResourceID creates a unique resource ID for a user
func (g *GatewayClient) generateResourceID(userID int, displayName string) string {
	// Clean display name for use in resource
	cleaned := strings.ToLower(displayName)
	cleaned = strings.ReplaceAll(cleaned, " ", "_")
	cleaned = strings.ReplaceAll(cleaned, "@", "_")

	// Format: user_ID_name
	return fmt.Sprintf("user_%d_%s", userID, cleaned)
}

// extractUserIDFromMessage attempts to extract user ID from admin's reply
func (g *GatewayClient) extractUserIDFromMessage(body string) int {
	// Look for patterns like "user_123" or "@user_123" or reply context
	// This is a simplified version - in production, you'd track conversation threads

	// For now, admin should reply with @user_ID format
	if strings.Contains(body, "@user_") {
		var userID int
		_, err := fmt.Sscanf(body, "%*s@user_%d", &userID)
		if err == nil {
			return userID
		}
	}

	return 0
}


// Close closes the gateway connection
func (g *GatewayClient) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.session != nil {
		// Send unavailable presence
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_ = g.session.Send(ctx, stanza.Presence{Type: stanza.UnavailablePresence}.Wrap(nil))

		err := g.session.Close()
		g.session = nil
		g.connected = false
		log.Println("Gateway: Connection closed")
		return err
	}

	g.connected = false
	return nil
}

// IsConnected returns true if the gateway is connected
func (g *GatewayClient) IsConnected() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.connected && g.session != nil
}
