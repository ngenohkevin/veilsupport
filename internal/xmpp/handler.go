package xmpp

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"veilsupport/internal/config"
)

type ChatService interface {
	SaveMessage(ctx context.Context, sessionID, fromJID, toJID, content, messageType string) error
	GetOrCreateSession(ctx context.Context, userEmail string) (string, error)
	BroadcastToWebSocket(userEmail, message string) error
}

type MessageHandler struct {
	xmppManager *XMPPManager
	chatService ChatService
	config      *config.Config
}

func NewMessageHandler(xmppManager *XMPPManager, chatService ChatService) *MessageHandler {
	return &MessageHandler{
		xmppManager: xmppManager,
		chatService: chatService,
		config:      xmppManager.config,
	}
}

func (h *MessageHandler) ProcessUserMessage(userEmail, message string) error {
	ctx := context.Background()

	// Get or create chat session for user
	sessionID, err := h.chatService.GetOrCreateSession(ctx, userEmail)
	if err != nil {
		return fmt.Errorf("failed to get/create session: %w", err)
	}

	// Generate user JID
	userJID := h.GenerateUserJID(userEmail)

	// Create user XMPP session if it doesn't exist
	if !h.xmppManager.HasUserSession(userJID) {
		_, err = h.xmppManager.CreateUserSession(userJID)
		if err != nil {
			return fmt.Errorf("failed to create user session: %w", err)
		}
	}

	// Send message to admin via XMPP
	err = h.xmppManager.SendMessage(userJID, h.config.XMPP.Admin, message)
	if err != nil {
		log.Printf("Failed to send XMPP message to admin: %v", err)
		// Don't return error here - we still want to save to database
	}

	// Save message to database
	err = h.chatService.SaveMessage(ctx, sessionID, userJID, h.config.XMPP.Admin, message, "user")
	if err != nil {
		return fmt.Errorf("failed to save user message: %w", err)
	}

	return nil
}

func (h *MessageHandler) ProcessAdminMessage(targetUser, message string) error {
	ctx := context.Background()

	// Get or create chat session for user
	sessionID, err := h.chatService.GetOrCreateSession(ctx, targetUser)
	if err != nil {
		return fmt.Errorf("failed to get/create session: %w", err)
	}

	// Generate user JID
	userJID := h.GenerateUserJID(targetUser)

	// Send message to user via XMPP (if they have an active session)
	if h.xmppManager.HasUserSession(userJID) {
		err = h.xmppManager.SendMessage(h.config.XMPP.Admin, userJID, message)
		if err != nil {
			log.Printf("Failed to send XMPP message to user: %v", err)
			// Continue to save and broadcast via WebSocket
		}
	}

	// Save message to database
	err = h.chatService.SaveMessage(ctx, sessionID, h.config.XMPP.Admin, userJID, message, "admin")
	if err != nil {
		return fmt.Errorf("failed to save admin message: %w", err)
	}

	// Broadcast to WebSocket for real-time delivery
	err = h.chatService.BroadcastToWebSocket(targetUser, message)
	if err != nil {
		log.Printf("Failed to broadcast message via WebSocket: %v", err)
		// Don't return error - message is saved
	}

	return nil
}

func (h *MessageHandler) StartListening(ctx context.Context) {
	// Start XMPP message listening
	h.xmppManager.StartListening(ctx)

	// Process incoming messages
	go h.processIncomingMessages(ctx)
}

func (h *MessageHandler) processIncomingMessages(ctx context.Context) {
	incoming := h.xmppManager.HandleIncoming()
	
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-incoming:
			if !ok {
				// Channel closed
				return
			}
			
			h.handleIncomingXMPPMessage(msg)
		}
	}
}

func (h *MessageHandler) handleIncomingXMPPMessage(msg XMPPMessage) {
	// Determine if message is from admin or user
	if msg.From == h.config.XMPP.Admin {
		// Admin message - extract target user from message or JID
		targetUser := h.extractUserEmailFromJID(msg.To)
		if targetUser != "" {
			err := h.ProcessAdminMessage(targetUser, msg.Body)
			if err != nil {
				log.Printf("Failed to process admin message: %v", err)
			}
		}
	} else {
		// User message
		userEmail := h.extractUserEmailFromJID(msg.From)
		if userEmail != "" {
			err := h.ProcessUserMessage(userEmail, msg.Body)
			if err != nil {
				log.Printf("Failed to process user message: %v", err)
			}
		}
	}
}

func (h *MessageHandler) GenerateUserJID(email string) string {
	// Convert email to valid XMPP JID
	// Replace special characters with underscores
	cleanEmail := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(email, "_")
	return fmt.Sprintf("user_%s@%s", cleanEmail, h.config.XMPP.Domain)
}

func (h *MessageHandler) extractUserEmailFromJID(jid string) string {
	// Extract user email from JID
	// Reverse the process of GenerateUserJID
	if !strings.Contains(jid, "@") {
		return ""
	}

	parts := strings.Split(jid, "@")
	if len(parts) != 2 {
		return ""
	}

	localPart := parts[0]
	if !strings.HasPrefix(localPart, "user_") {
		return ""
	}

	// Remove "user_" prefix
	cleanPart := strings.TrimPrefix(localPart, "user_")
	
	// This is a simplified reverse conversion
	// In production, you'd want a more robust mapping
	return strings.ReplaceAll(cleanPart, "_", "@")
}

func (h *MessageHandler) Shutdown() {
	// Disconnect admin first
	h.xmppManager.DisconnectAdmin()
	
	// Then shutdown the manager
	h.xmppManager.Shutdown()
}