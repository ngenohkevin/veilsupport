package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ngenohkevin/veilsupport/internal/db"
	"github.com/ngenohkevin/veilsupport/internal/ws"
	"github.com/ngenohkevin/veilsupport/internal/xmpp"
)

type ChatService struct {
	db   *db.DB
	xmpp *xmpp.XMPPClient
	ws   *ws.Manager
}

func NewChatService(database *db.DB, xmppClient *xmpp.XMPPClient, wsManager *ws.Manager) *ChatService {
	return &ChatService{
		db:   database,
		xmpp: xmppClient,
		ws:   wsManager,
	}
}

func (s *ChatService) SendMessage(userID int, content string) error {
	// Get user
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	
	// Save to database first (always save even if XMPP fails)
	_, err = s.db.SaveMessage(userID, content, "user")
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	
	// Try to send via XMPP if connected
	if s.xmpp != nil && s.xmpp.IsConnected() {
		adminJID := os.Getenv("XMPP_ADMIN_JID")
		if adminJID == "" {
			log.Println("XMPP_ADMIN_JID not configured")
			return nil // Don't fail the whole operation
		}
		
		// Format message with user email for context
		message := fmt.Sprintf("[User: %s] %s", user.Email, content)
		
		// Try to send via XMPP
		err = s.xmpp.SendMessage(adminJID, message)
		if err != nil {
			// Try the simple send method as fallback
			log.Printf("Regular XMPP send failed: %v, trying simple method...", err)
			err = s.xmpp.SendMessageSimple(adminJID, message)
			if err != nil {
				log.Printf("XMPP send failed (both methods): %v", err)
				// Don't return error - message is saved in DB
			} else {
				log.Printf("XMPP message sent via simple method to %s", adminJID)
			}
		} else {
			log.Printf("XMPP message sent to %s", adminJID)
		}
	} else {
		log.Println("XMPP not connected - message saved to database only")
	}
	
	return nil
}

func (s *ChatService) HandleAdminReply(xmppMsg xmpp.XMPPMessage) error {
	// Extract user JID from message - admin replies are sent TO the user
	userJID := xmppMsg.To
	
	// Find user
	user, err := s.db.GetUserByJID(userJID)
	if err != nil {
		return fmt.Errorf("failed to find user by JID: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found for JID: %s", userJID)
	}
	
	// Save to database
	_, err = s.db.SaveMessage(user.ID, xmppMsg.Body, "admin")
	if err != nil {
		return fmt.Errorf("failed to save admin message: %w", err)
	}
	
	// Send via WebSocket if user is connected
	if s.ws != nil {
		wsMsg := map[string]string{
			"type":    "message",
			"content": xmppMsg.Body,
			"from":    "admin",
		}
		
		data, err := json.Marshal(wsMsg)
		if err != nil {
			return fmt.Errorf("failed to marshal WebSocket message: %w", err)
		}
		
		s.ws.SendToUser(user.ID, data)
		log.Printf("Admin reply sent to user %s via WebSocket", user.Email)
	}
	
	return nil
}

func (s *ChatService) StartXMPPListener(ctx context.Context) {
	if s.xmpp == nil {
		log.Println("XMPP client not initialized, skipping listener")
		return
	}
	
	messages := make(chan xmpp.XMPPMessage, 100)
	errorChan := make(chan error, 10)
	
	// Start XMPP listener in goroutine
	go func() {
		err := s.xmpp.Listen(ctx, messages, errorChan)
		if err != nil {
			log.Printf("XMPP listener error: %v", err)
		}
	}()
	
	log.Println("XMPP listener started")
	
	// Handle messages and errors
	for {
		select {
		case msg := <-messages:
			log.Printf("Received XMPP message from %s to %s: %s", msg.From, msg.To, msg.Body)
			if err := s.HandleAdminReply(msg); err != nil {
				log.Printf("Error handling XMPP message: %v", err)
			}
		case err := <-errorChan:
			log.Printf("XMPP error: %v", err)
		case <-ctx.Done():
			log.Println("XMPP listener stopping")
			return
		}
	}
}

func (s *ChatService) GetUserMessages(userID int) ([]db.Message, error) {
	messages, err := s.db.GetUserMessages(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user messages: %w", err)
	}
	return messages, nil
}
