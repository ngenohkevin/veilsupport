package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ngenohkevin/veilsupport/internal/db"
	"github.com/ngenohkevin/veilsupport/internal/ws"
	"github.com/ngenohkevin/veilsupport/internal/xmpp"
)

// GatewayService handles chat using the XMPP gateway approach
type GatewayService struct {
	db      *db.DB
	gateway *xmpp.GatewayClient
	ws      *ws.Manager
}

// NewGatewayService creates a new gateway-based chat service
func NewGatewayService(database *db.DB, wsManager *ws.Manager) *GatewayService {
	// Get admin JIDs from environment
	adminJIDsStr := os.Getenv("XMPP_ADMIN_JIDS")
	if adminJIDsStr == "" {
		adminJIDsStr = os.Getenv("XMPP_ADMIN_JID") // Fallback to single admin
	}
	
	// Parse multiple admin JIDs
	adminJIDs := strings.Split(adminJIDsStr, ",")
	for i := range adminJIDs {
		adminJIDs[i] = strings.TrimSpace(adminJIDs[i])
	}
	
	// Get bot credentials
	botJID := os.Getenv("XMPP_BOT_JID")
	if botJID == "" {
		// Fallback to connection JID
		botJID = os.Getenv("XMPP_CONNECTION_JID")
	}
	
	botPassword := os.Getenv("XMPP_BOT_PASSWORD")
	if botPassword == "" {
		// Fallback to connection password
		botPassword = os.Getenv("XMPP_CONNECTION_PASSWORD")
	}
	
	xmppServer := os.Getenv("XMPP_SERVER")
	
	// Create gateway client
	gateway := xmpp.NewGatewayClient(botJID, botPassword, xmppServer, adminJIDs)
	
	return &GatewayService{
		db:      database,
		gateway: gateway,
		ws:      wsManager,
	}
}

// Connect initializes the gateway connection
func (s *GatewayService) Connect(ctx context.Context) error {
	err := s.gateway.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect gateway: %w", err)
	}
	
	log.Println("Gateway: Connected successfully")
	return nil
}

// RegisterUser registers a web user with the gateway
func (s *GatewayService) RegisterUser(userID int) error {
	// Get user from database
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	
	// Extract display name from email or use email
	displayName := user.Email
	if atIndex := strings.Index(user.Email, "@"); atIndex > 0 {
		displayName = user.Email[:atIndex]
	}
	
	// Register with gateway
	resourceID := s.gateway.RegisterUser(userID, user.Email, displayName)
	
	log.Printf("Gateway: Registered user %s as %s", user.Email, resourceID)
	return nil
}

// SendMessage sends a message from a web user through the gateway
func (s *GatewayService) SendMessage(userID int, content string, attachments []string) error {
	// Ensure user is registered with gateway
	err := s.RegisterUser(userID)
	if err != nil {
		log.Printf("Gateway: Failed to register user %d: %v", userID, err)
	}
	
	// Save to database first
	_, err = s.db.SaveMessage(userID, content, "user")
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	
	// Send through gateway if connected
	if s.gateway != nil && s.gateway.IsConnected() {
		err = s.gateway.SendUserMessage(userID, content, attachments)
		if err != nil {
			log.Printf("Gateway: Failed to send message via XMPP: %v", err)
			// Don't fail - message is saved in DB
		} else {
			log.Printf("Gateway: Message sent from user %d", userID)
		}
	} else {
		log.Println("Gateway: Not connected - message saved to database only")
	}
	
	// Update user online status
	if s.gateway != nil && s.gateway.IsConnected() {
		s.gateway.SetUserOnline(userID, true)
	}
	
	return nil
}

// HandleAdminReply processes a reply from admin through the gateway
func (s *GatewayService) HandleAdminReply(from, body string) error {
	// Let gateway parse the message and determine target user
	gwMsg, err := s.gateway.HandleAdminReply(from, body)
	if err != nil {
		return fmt.Errorf("failed to handle admin reply: %w", err)
	}
	
	// Save to database
	_, err = s.db.SaveMessage(gwMsg.UserID, gwMsg.Body, "admin")
	if err != nil {
		return fmt.Errorf("failed to save admin message: %w", err)
	}
	
	// Send via WebSocket to user if connected
	if s.ws != nil {
		wsMsg := map[string]interface{}{
			"type":      "message",
			"content":   gwMsg.Body,
			"from":      "admin",
			"timestamp": gwMsg.Timestamp,
		}
		
		if len(gwMsg.Attachments) > 0 {
			wsMsg["attachments"] = gwMsg.Attachments
		}
		
		data, err := json.Marshal(wsMsg)
		if err != nil {
			return fmt.Errorf("failed to marshal WebSocket message: %w", err)
		}
		
		s.ws.SendToUser(gwMsg.UserID, data)
		log.Printf("Gateway: Admin reply sent to user %s via WebSocket", gwMsg.UserEmail)
	}
	
	return nil
}

// SetUserOnline updates user's online status
func (s *GatewayService) SetUserOnline(userID int, online bool) error {
	if s.gateway != nil && s.gateway.IsConnected() {
		return s.gateway.SetUserOnline(userID, online)
	}
	return nil
}

// UploadFile handles file uploads from web users
func (s *GatewayService) UploadFile(userID int, filename string, data []byte) (string, error) {
	// In production, you'd store this in S3 or similar
	// For now, store locally
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "/tmp/veilsupport/uploads"
	}
	
	// Create upload directory if it doesn't exist
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}
	
	// Generate unique filename
	uniqueFilename := fmt.Sprintf("%d_%d_%s", userID, time.Now().Unix(), filename)
	filepath := fmt.Sprintf("%s/%s", uploadDir, uniqueFilename)
	
	// Write file
	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	
	// Return URL (in production, this would be a public URL)
	url := fmt.Sprintf("/uploads/%s", uniqueFilename)
	
	log.Printf("Gateway: File uploaded for user %d: %s", userID, url)
	return url, nil
}

// GetUserMessages retrieves message history for a user
func (s *GatewayService) GetUserMessages(userID int) ([]db.Message, error) {
	messages, err := s.db.GetUserMessages(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user messages: %w", err)
	}
	return messages, nil
}

// StartListener starts listening for XMPP messages
func (s *GatewayService) StartListener(ctx context.Context) {
	if s.gateway == nil {
		log.Println("Gateway: No gateway configured, skipping listener")
		return
	}
	
	// This would listen for incoming admin messages
	// For now, it's a placeholder - real implementation would use
	// the gateway's message handling
	log.Println("Gateway: Listener started (placeholder)")
	
	<-ctx.Done()
	log.Println("Gateway: Listener stopped")
}

// Close closes the gateway connection
func (s *GatewayService) Close() error {
	if s.gateway != nil {
		return s.gateway.Close()
	}
	return nil
}
