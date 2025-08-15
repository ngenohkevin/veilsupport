package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/veilsupport/internal/auth"
	"github.com/ngenohkevin/veilsupport/internal/chat"
	"github.com/ngenohkevin/veilsupport/internal/db"
	"github.com/ngenohkevin/veilsupport/internal/handlers"
	"github.com/ngenohkevin/veilsupport/internal/ws"
	"github.com/ngenohkevin/veilsupport/internal/xmpp"
)

func main() {
	// Load config from environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:pass@localhost/veilsupport"
		log.Println("Using default DATABASE_URL")
	}
	
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-this"
		log.Println("WARNING: Using default JWT_SECRET - change this in production!")
	}
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// XMPP configuration
	xmppServer := os.Getenv("XMPP_SERVER")
	if xmppServer == "" {
		xmppServer = "xmpp.server.com"
		log.Println("Using default XMPP_SERVER")
	}
	
	// XMPP connection credentials (for connecting to server)
	xmppConnectionJID := os.Getenv("XMPP_CONNECTION_JID")
	if xmppConnectionJID == "" {
		xmppConnectionJID = os.Getenv("XMPP_ADMIN_JID") // Fallback to admin JID
		if xmppConnectionJID == "" {
			xmppConnectionJID = "admin@xmpp.server.com"
			log.Println("Using default XMPP_CONNECTION_JID")
		}
	}
	
	xmppConnectionPassword := os.Getenv("XMPP_CONNECTION_PASSWORD")
	if xmppConnectionPassword == "" {
		xmppConnectionPassword = os.Getenv("XMPP_ADMIN_PASSWORD") // Fallback to admin password
		if xmppConnectionPassword == "" {
			xmppConnectionPassword = "admin-password"
			log.Println("Using default XMPP_CONNECTION_PASSWORD")
		}
	}
	
	// Log configuration (without sensitive data)
	log.Printf("Starting VeilSupport server with config:")
	log.Printf("  Port: %s", port)
	log.Printf("  XMPP Server: %s", xmppServer)
	log.Printf("  XMPP Connection JID: %s", xmppConnectionJID)
	
	// Initialize database
	database, err := db.New(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	
	// Initialize auth service
	authService := auth.NewAuthService(database, jwtSecret)
	
	// Initialize XMPP client
	xmppClient := xmpp.NewXMPPClient(xmppConnectionJID, xmppConnectionPassword, xmppServer)
	
	// Initialize WebSocket manager
	wsManager := ws.NewManager()
	
	// Initialize chat service
	chatService := chat.NewChatService(database, xmppClient, wsManager)
	
	// Initialize handlers
	h := handlers.NewHandlers(authService, chatService, wsManager)
	
	// Connect to XMPP server (optional - can fail gracefully)
	ctx := context.Background()
	if err := xmppClient.ConnectWithContext(ctx); err != nil {
		log.Printf("Warning: Failed to connect to XMPP server: %v", err)
		log.Println("Continuing without XMPP - messages will be saved to database only")
	} else {
		log.Println("Connected to XMPP server successfully")
		
		// Start XMPP listener in background
		go chatService.StartXMPPListener(ctx)
	}
	
	// Setup router
	r := gin.Default()
	
	// API routes
	api := r.Group("/api")
	{
		// Public endpoints
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		
		// Protected endpoints
		protected := api.Group("/")
		protected.Use(h.JWTMiddleware())
		{
			protected.POST("/send", h.SendMessage)
			protected.GET("/history", h.GetHistory)
			protected.GET("/ws", h.WebSocket)
		}
	}
	
	// Start server
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}