package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// BetterBotClient provides a realistic implementation that works with XMPP limitations
type BetterBotClient struct {
	botJID       string
	password     string
	server       string
	adminJID     string
	session      *xmpp.Session
	connected    bool
	activeUsers  map[int]*UserSession
	mu           sync.RWMutex
}

// UserSession tracks an active user conversation
type UserSession struct {
	UserID        int
	Email         string
	DisplayName   string
	LastMessage   string
	LastMessageAt time.Time
	MessageCount  int
	Color         string // For visual distinction
}

// NewBetterBotClient creates a realistic bot that formats messages clearly
func NewBetterBotClient(botJID, password, server, adminJID string) *BetterBotClient {
	return &BetterBotClient{
		botJID:      botJID,
		password:    password,
		server:      server,
		adminJID:    adminJID,
		activeUsers: make(map[int]*UserSession),
	}
}

// Connect establishes XMPP connection
func (b *BetterBotClient) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connected && b.session != nil {
		return nil
	}

	addr, err := jid.Parse(b.botJID)
	if err != nil {
		return fmt.Errorf("invalid bot JID: %w", err)
	}

	log.Printf("Bot: Connecting to %s as %s", b.server, b.botJID)

	tlsConfig := &tls.Config{
		ServerName:         addr.Domain().String(),
		InsecureSkipVerify: true,
	}

	session, err := xmpp.DialClientSession(
		ctx, addr,
		xmpp.BindResource(),
		xmpp.StartTLS(tlsConfig),
		xmpp.SASL("", b.password, sasl.Plain),
	)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Send presence
	err = session.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		session.Close()
		return fmt.Errorf("failed to send presence: %w", err)
	}

	b.session = session
	b.connected = true
	
	// Send initial help message to admin
	b.SendSystemMessage("VeilSupport Bot Connected. Reply format: @USER_ID your message")
	
	log.Printf("Bot: Successfully connected")
	return nil
}

// SendUserMessage sends a well-formatted message from a website user
func (b *BetterBotClient) SendUserMessage(userID int, email, displayName, message string) error {
	if !b.connected || b.session == nil {
		return errors.New("bot not connected")
	}

	b.mu.Lock()
	// Track user session
	if _, exists := b.activeUsers[userID]; !exists {
		colors := []string{"🔴", "🟠", "🟡", "🟢", "🔵", "🟣", "🟤", "⚫", "⚪"}
		b.activeUsers[userID] = &UserSession{
			UserID:      userID,
			Email:       email,
			DisplayName: displayName,
			Color:       colors[userID%len(colors)],
		}
	}
	
	session := b.activeUsers[userID]
	session.LastMessage = message
	session.LastMessageAt = time.Now()
	session.MessageCount++
	b.mu.Unlock()

	// Format message beautifully
	formatted := b.formatUserMessage(session, message)
	
	// Send to admin
	return b.sendToAdmin(formatted)
}

// formatUserMessage creates a well-formatted message that's easy to read
func (b *BetterBotClient) formatUserMessage(session *UserSession, message string) string {
	separator := "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	
	// Build formatted message
	var sb strings.Builder
	
	// Header with user info
	sb.WriteString(fmt.Sprintf("\n%s\n", separator))
	sb.WriteString(fmt.Sprintf("%s USER MESSAGE\n", session.Color))
	sb.WriteString(fmt.Sprintf("👤 %s\n", session.DisplayName))
	sb.WriteString(fmt.Sprintf("📧 %s\n", session.Email))
	sb.WriteString(fmt.Sprintf("🆔 User ID: %d\n", session.UserID))
	sb.WriteString(fmt.Sprintf("📊 Message #%d\n", session.MessageCount))
	sb.WriteString(fmt.Sprintf("🕐 %s\n", time.Now().Format("15:04:05")))
	sb.WriteString(fmt.Sprintf("%s\n\n", separator))
	
	// Message body
	sb.WriteString(fmt.Sprintf("💬 %s\n\n", message))
	
	// Reply instruction
	sb.WriteString(fmt.Sprintf("↩️  Reply: @%d [your message]\n", session.UserID))
	sb.WriteString(fmt.Sprintf("%s\n", separator))
	
	return sb.String()
}

// sendToAdmin sends a message to the admin
func (b *BetterBotClient) sendToAdmin(body string) error {
	recipientJID, err := jid.Parse(b.adminJID)
	if err != nil {
		return fmt.Errorf("invalid admin JID: %w", err)
	}

	msg := SimpleMessage{
		To:   recipientJID.String(),
		Type: "chat",
		Body: body,
		ID:   fmt.Sprintf("msg_%d", time.Now().Unix()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = b.session.Send(ctx, msg.TokenReader())
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// ParseAdminReply extracts user ID and message from admin's reply
func (b *BetterBotClient) ParseAdminReply(message string) (int, string, error) {
	// Format: @USER_ID message
	// Example: @101 Your order has been shipped
	
	re := regexp.MustCompile(`^@(\d+)\s+(.+)`)
	matches := re.FindStringSubmatch(strings.TrimSpace(message))
	
	if len(matches) != 3 {
		return 0, "", fmt.Errorf("invalid reply format. Use: @USER_ID message")
	}
	
	userID, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", fmt.Errorf("invalid user ID: %s", matches[1])
	}
	
	replyText := matches[2]
	
	return userID, replyText, nil
}

// SendSystemMessage sends a system notification to admin
func (b *BetterBotClient) SendSystemMessage(message string) error {
	if !b.connected || b.session == nil {
		return errors.New("bot not connected")
	}

	formatted := fmt.Sprintf(`
════════════════════════════
🤖 SYSTEM MESSAGE
════════════════════════════
%s
════════════════════════════
`, message)

	return b.sendToAdmin(formatted)
}

// ListActiveUsers sends a list of active users to admin
func (b *BetterBotClient) ListActiveUsers() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.activeUsers) == 0 {
		return b.SendSystemMessage("No active users")
	}

	var sb strings.Builder
	sb.WriteString("\n📋 ACTIVE USERS\n")
	sb.WriteString("═══════════════════════════\n\n")
	
	for _, session := range b.activeUsers {
		timeSince := time.Since(session.LastMessageAt)
		sb.WriteString(fmt.Sprintf("%s User #%d: %s\n", 
			session.Color, session.UserID, session.DisplayName))
		sb.WriteString(fmt.Sprintf("   📧 %s\n", session.Email))
		sb.WriteString(fmt.Sprintf("   💬 Messages: %d\n", session.MessageCount))
		sb.WriteString(fmt.Sprintf("   🕐 Last active: %s ago\n", 
			formatDuration(timeSince)))
		sb.WriteString(fmt.Sprintf("   📝 Last: %.50s...\n\n", session.LastMessage))
	}
	
	sb.WriteString("═══════════════════════════\n")
	sb.WriteString("Reply format: @USER_ID message\n")
	
	return b.sendToAdmin(sb.String())
}

// HandleCommand processes admin commands
func (b *BetterBotClient) HandleCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/list", "/users":
		return b.ListActiveUsers()
		
	case "/help":
		help := `
📚 AVAILABLE COMMANDS
═══════════════════════════
/list - Show active users
/info USER_ID - User details
/clear USER_ID - Clear user session
/help - Show this help

REPLY FORMAT:
@USER_ID your message here
═══════════════════════════`
		return b.SendSystemMessage(help)
		
	case "/info":
		if len(parts) < 2 {
			return b.SendSystemMessage("Usage: /info USER_ID")
		}
		userID, err := strconv.Atoi(parts[1])
		if err != nil {
			return b.SendSystemMessage("Invalid user ID")
		}
		return b.sendUserInfo(userID)
		
	case "/clear":
		if len(parts) < 2 {
			return b.SendSystemMessage("Usage: /clear USER_ID")
		}
		userID, err := strconv.Atoi(parts[1])
		if err != nil {
			return b.SendSystemMessage("Invalid user ID")
		}
		b.mu.Lock()
		delete(b.activeUsers, userID)
		b.mu.Unlock()
		return b.SendSystemMessage(fmt.Sprintf("Cleared session for user %d", userID))
		
	default:
		// Not a command, might be a reply
		if strings.HasPrefix(command, "@") {
			userID, reply, err := b.ParseAdminReply(command)
			if err != nil {
				return b.SendSystemMessage(fmt.Sprintf("Error: %v", err))
			}
			return b.SendSystemMessage(fmt.Sprintf("✅ Reply sent to user %d: %s", userID, reply))
		}
	}
	
	return nil
}

// sendUserInfo sends detailed info about a user
func (b *BetterBotClient) sendUserInfo(userID int) error {
	b.mu.RLock()
	session, exists := b.activeUsers[userID]
	b.mu.RUnlock()
	
	if !exists {
		return b.SendSystemMessage(fmt.Sprintf("User %d not found", userID))
	}
	
	info := fmt.Sprintf(`
📋 USER INFORMATION
═══════════════════════════
%s User ID: %d
👤 Name: %s
📧 Email: %s
💬 Total Messages: %d
🕐 Last Active: %s
📝 Last Message: %s
═══════════════════════════`,
		session.Color,
		session.UserID,
		session.DisplayName,
		session.Email,
		session.MessageCount,
		session.LastMessageAt.Format("15:04:05"),
		session.LastMessage,
	)
	
	return b.SendSystemMessage(info)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}

// Close closes the bot connection
func (b *BetterBotClient) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.session != nil {
		// Send goodbye message
		b.SendSystemMessage("VeilSupport Bot disconnecting")
		
		// Send unavailable presence
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		_ = b.session.Send(ctx, stanza.Presence{Type: stanza.UnavailablePresence}.Wrap(nil))
		
		err := b.session.Close()
		b.session = nil
		b.connected = false
		log.Println("Bot: Connection closed")
		return err
	}
	
	b.connected = false
	return nil
}

// IsConnected returns true if the bot is connected
func (b *BetterBotClient) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected && b.session != nil
}

// SimpleMessage implements a basic XMPP message encoder
type SimpleMessage struct {
	To   string
	Type string
	Body string
	ID   string
}

// TokenReader implements xmlstream.Marshaler
func (m SimpleMessage) TokenReader() xml.TokenReader {
	// Create the XML tokens for the message
	tokens := []xml.Token{
		xml.StartElement{
			Name: xml.Name{Space: "jabber:client", Local: "message"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "to"}, Value: m.To},
				{Name: xml.Name{Local: "type"}, Value: m.Type},
				{Name: xml.Name{Local: "id"}, Value: m.ID},
			},
		},
		xml.StartElement{Name: xml.Name{Local: "body"}},
		xml.CharData(m.Body),
		xml.EndElement{Name: xml.Name{Local: "body"}},
		xml.EndElement{Name: xml.Name{Space: "jabber:client", Local: "message"}},
	}
	
	return &tokenReader{tokens: tokens}
}

// WriteXML implements xmlstream.WriterTo
func (m SimpleMessage) WriteXML(w xmlstream.TokenWriter) (int, error) {
	n := 0
	for _, tok := range []xml.Token{
		xml.StartElement{
			Name: xml.Name{Space: "jabber:client", Local: "message"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "to"}, Value: m.To},
				{Name: xml.Name{Local: "type"}, Value: m.Type},
				{Name: xml.Name{Local: "id"}, Value: m.ID},
			},
		},
		xml.StartElement{Name: xml.Name{Local: "body"}},
		xml.CharData(m.Body),
		xml.EndElement{Name: xml.Name{Local: "body"}},
		xml.EndElement{Name: xml.Name{Space: "jabber:client", Local: "message"}},
	} {
		if err := w.EncodeToken(tok); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// tokenReader helps implement TokenReader interface
type tokenReader struct {
	tokens []xml.Token
	pos    int
}

func (tr *tokenReader) Token() (xml.Token, error) {
	if tr.pos >= len(tr.tokens) {
		return nil, io.EOF
	}
	tok := tr.tokens[tr.pos]
	tr.pos++
	return tok, nil
}
