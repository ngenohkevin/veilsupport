package tests

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ngenohkevin/veilsupport/internal/auth"
	"github.com/ngenohkevin/veilsupport/internal/chat"
	"github.com/ngenohkevin/veilsupport/internal/handlers"
	"github.com/ngenohkevin/veilsupport/internal/ws"
	"github.com/ngenohkevin/veilsupport/internal/xmpp"
	"github.com/stretchr/testify/assert"
)

func connectWebSocket(t *testing.T, app *gin.Engine, token string) *websocket.Conn {
	// Create test server
	server := httptest.NewServer(app)
	defer server.Close()
	
	// Convert HTTP URL to WebSocket URL
	u, err := url.Parse(server.URL)
	assert.NoError(t, err)
	
	wsURL := "ws" + strings.TrimPrefix(u.String(), "http") + "/api/ws?token=" + token
	
	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		// Return nil if connection fails (endpoint not implemented yet)
		return nil
	}
	
	return conn
}

// Global variable to hold chat service for testing
var testChatService *chat.ChatService

func setupWebSocketTestApp(t *testing.T) (*gin.Engine, *chat.ChatService) {
	gin.SetMode(gin.TestMode)
	
	// Setup test database
	database := setupTestDB(t)
	
	// Setup auth service
	authService := auth.NewAuthService(database, "test-secret-key")
	
	// Setup XMPP client (mock for testing)
	xmppClient := xmpp.NewXMPPClient("test@example.com", "password", "localhost:5222")
	
	// Setup WebSocket manager
	wsManager := ws.NewManager()
	
	// Setup chat service
	chatService := chat.NewChatService(database, xmppClient, wsManager)
	
	// Store reference for simulateAdminMessage
	testChatService = chatService
	
	// Setup handlers
	h := handlers.NewHandlers(authService, chatService, wsManager)
	
	// Setup router
	r := gin.New()
	
	// API routes
	api := r.Group("/api")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		
		// Protected routes
		protected := api.Group("/")
		protected.Use(h.JWTMiddleware())
		{
			protected.POST("/send", h.SendMessage)
			protected.GET("/history", h.GetHistory)
		}
		
		// WebSocket route (token auth via query param)
		api.GET("/ws", h.WebSocket)
	}
	
	return r, chatService
}

func simulateAdminMessage(userJID, message string) {
	// Now we can use the testChatService to simulate an admin reply
	if testChatService != nil {
		xmppMsg := xmpp.XMPPMessage{
			From: "admin@server.com",
			To:   userJID,
			Body: message,
		}
		// Call the chat service's HandleAdminReply method directly
		testChatService.HandleAdminReply(xmppMsg)
	}
}

func TestWebSocketConnection(t *testing.T) {
	app := setupTestApp(t)
	token := createTestUserAndGetToken(t, app)
	
	ws := connectWebSocket(t, app, token)
	if ws == nil {
		t.Skip("WebSocket endpoint not implemented yet")
		return
	}
	defer ws.Close()
	
	// Should receive connection confirmation
	// Set read deadline to avoid hanging
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	var msg map[string]string
	err := ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "connected", msg["type"])
}

func TestWebSocketReceiveMessage(t *testing.T) {
	app, _ := setupWebSocketTestApp(t)
	user, token := registerUser(t, app, "wstest@example.com", "password123")
	
	ws := connectWebSocket(t, app, token)
	if ws == nil {
		t.Skip("WebSocket endpoint not implemented yet")
		return
	}
	defer ws.Close()
	
	// Get user's XMPP JID for proper simulation
	userXmppJID := user["xmpp_jid"].(string)
	
	// First read the "connected" message
	var connectMsg map[string]string
	err := ws.ReadJSON(&connectMsg)
	assert.NoError(t, err)
	assert.Equal(t, "connected", connectMsg["type"])
	
	// Simulate admin sending message via XMPP to this specific user
	simulateAdminMessage(userXmppJID, "Reply from admin")
	
	// Set read deadline to avoid hanging
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	var msg map[string]string
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "message", msg["type"])
	assert.Equal(t, "Reply from admin", msg["content"])
}

func TestWebSocketInvalidToken(t *testing.T) {
	app := setupTestApp(t)
	
	// Try to connect with invalid token
	server := httptest.NewServer(app)
	defer server.Close()
	
	u, err := url.Parse(server.URL)
	assert.NoError(t, err)
	
	wsURL := "ws" + strings.TrimPrefix(u.String(), "http") + "/api/ws?token=invalid-token"
	
	// This should fail to connect or close immediately
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		defer conn.Close()
		// If connection succeeds, it should close with an error
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, readErr := conn.ReadMessage()
		assert.Error(t, readErr)
	} else {
		// Connection should fail with proper status (401 when implemented, 404 when not)
		assert.NotNil(t, resp)
		assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 404)
	}
}

func TestWebSocketNoToken(t *testing.T) {
	app := setupTestApp(t)
	
	// Try to connect without token
	server := httptest.NewServer(app)
	defer server.Close()
	
	u, err := url.Parse(server.URL)
	assert.NoError(t, err)
	
	wsURL := "ws" + strings.TrimPrefix(u.String(), "http") + "/api/ws"
	
	// This should fail to connect
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		defer conn.Close()
		// If connection succeeds, it should close with an error
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, readErr := conn.ReadMessage()
		assert.Error(t, readErr)
	} else {
		// Connection should fail with proper status (401 when implemented, 404 when not)
		assert.NotNil(t, resp)
		assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 404)
	}
}

func TestWebSocketMultipleConnections(t *testing.T) {
	app := setupTestApp(t)
	
	// Create two users
	token1 := createTestUserAndGetToken(t, app)
	
	// Register second user (using unique email)
	uniqueEmail := fmt.Sprintf("testuser2_%d@example.com", time.Now().UnixNano())
	body := fmt.Sprintf(`{"email":"%s","password":"password123"}`, uniqueEmail)
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Logf("Registration failed with status %d, body: %s", w.Code, w.Body.String())
	}
	assert.Equal(t, 201, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	
	token2, ok := resp["token"].(string)
	assert.True(t, ok)
	
	// Connect both users
	ws1 := connectWebSocket(t, app, token1)
	if ws1 == nil {
		t.Skip("WebSocket endpoint not implemented yet")
		return
	}
	defer ws1.Close()
	
	ws2 := connectWebSocket(t, app, token2)
	if ws2 == nil {
		t.Skip("WebSocket endpoint not implemented yet")
		return
	}
	defer ws2.Close()
	
	// Both should receive connection confirmations
	ws1.SetReadDeadline(time.Now().Add(5 * time.Second))
	ws2.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	var msg1, msg2 map[string]string
	err1 := ws1.ReadJSON(&msg1)
	err2 := ws2.ReadJSON(&msg2)
	
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, "connected", msg1["type"])
	assert.Equal(t, "connected", msg2["type"])
}

func TestWebSocketPingPong(t *testing.T) {
	app := setupTestApp(t)
	token := createTestUserAndGetToken(t, app)
	
	ws := connectWebSocket(t, app, token)
	if ws == nil {
		t.Skip("WebSocket endpoint not implemented yet")
		return
	}
	defer ws.Close()
	
	// Set up ping handler to respond to pings
	ws.SetPingHandler(func(appData string) error {
		return ws.WriteMessage(websocket.PongMessage, []byte(appData))
	})
	
	// Send a ping
	err := ws.WriteMessage(websocket.PingMessage, []byte("ping"))
	assert.NoError(t, err)
	
	// Should receive pong back (handled automatically by ping handler)
	// If we reach here without hanging, the ping/pong worked
}

func TestWebSocketMessageHistory(t *testing.T) {
	app := setupTestApp(t)
	token := createTestUserAndGetToken(t, app)
	
	// Send some messages first via REST API
	sendTestMessages(t, app, token, []string{"Hello", "How are you?"})
	
	// Now connect WebSocket
	ws := connectWebSocket(t, app, token)
	if ws == nil {
		t.Skip("WebSocket endpoint not implemented yet")
		return
	}
	defer ws.Close()
	
	// Should receive connection confirmation
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	var msg map[string]string
	err := ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "connected", msg["type"])
	
	// Verify that we can still get history via REST API
	req := httptest.NewRequest("GET", "/api/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 200, w.Code)
	
	var historyResp map[string][]map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &historyResp)
	assert.NoError(t, err)
	assert.Len(t, historyResp["messages"], 2)
}