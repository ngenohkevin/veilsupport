package xmpp

import (
	"context"
	"log"

	"veilsupport/internal/chat"
	"veilsupport/internal/config"
	"veilsupport/internal/database"
)

type Service struct {
	xmppManager    *XMPPManager
	messageHandler *MessageHandler
	chatService    *chat.Service
	config         *config.Config
}

func NewService(dbService *database.Service, wsManager chat.WSManager, cfg *config.Config) *Service {
	// Create XMPP manager
	xmppManager := NewXMPPManager(cfg)
	
	// Create chat service
	chatService := chat.NewService(dbService, wsManager)
	
	// Create message handler
	messageHandler := NewMessageHandler(xmppManager, chatService)
	
	return &Service{
		xmppManager:    xmppManager,
		messageHandler: messageHandler,
		chatService:    chatService,
		config:         cfg,
	}
}

func (s *Service) Start(ctx context.Context) error {
	log.Println("Starting XMPP service...")
	
	// Connect admin to XMPP server
	err := s.xmppManager.ConnectAdmin()
	if err != nil {
		return err
	}
	log.Println("Admin connected to XMPP server")
	
	// Start message handling
	s.messageHandler.StartListening(ctx)
	log.Println("XMPP message handler started")
	
	return nil
}

func (s *Service) Stop() {
	log.Println("Stopping XMPP service...")
	s.messageHandler.Shutdown()
	log.Println("XMPP service stopped")
}

func (s *Service) SendUserMessage(userEmail, message string) error {
	return s.messageHandler.ProcessUserMessage(userEmail, message)
}

func (s *Service) SendAdminMessage(targetUser, message string) error {
	return s.messageHandler.ProcessAdminMessage(targetUser, message)
}

func (s *Service) GetChatService() *chat.Service {
	return s.chatService
}

func (s *Service) GetXMPPManager() *XMPPManager {
	return s.xmppManager
}