package tests

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/veilsupport/internal/auth"
	"github.com/ngenohkevin/veilsupport/internal/chat"
	"github.com/ngenohkevin/veilsupport/internal/handlers"
	"github.com/ngenohkevin/veilsupport/internal/ws"
	"github.com/ngenohkevin/veilsupport/internal/xmpp"
	"github.com/stretchr/testify/assert"
)

func setupTestApp(t *testing.T) *gin.Engine {
	// Set Gin to test mode
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
	
	return r
}

func createTestUserAndGetToken(t *testing.T, app *gin.Engine) string {
	// Register a test user
	body := `{"email":"testuser@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 201, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	
	token, ok := resp["token"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, token)
	
	return token
}

func sendTestMessages(t *testing.T, app *gin.Engine, token string, messages []string) {
	for _, msg := range messages {
		body := fmt.Sprintf(`{"message":"%s"}`, msg)
		req := httptest.NewRequest("POST", "/api/send", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		
		assert.Equal(t, 200, w.Code)
	}
}

func TestRegisterEndpoint(t *testing.T) {
	app := setupTestApp(t)
	
	// Test successful registration
	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 201, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["token"])
	
	user, ok := resp["user"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "test@example.com", user["email"])
	assert.NotEmpty(t, user["xmpp_jid"])
}

func TestRegisterEndpointValidation(t *testing.T) {
	app := setupTestApp(t)
	
	testCases := []struct {
		name       string
		body       string
		expectCode int
	}{
		{
			name:       "missing email",
			body:       `{"password":"password123"}`,
			expectCode: 400,
		},
		{
			name:       "missing password",
			body:       `{"email":"test@example.com"}`,
			expectCode: 400,
		},
		{
			name:       "invalid email",
			body:       `{"email":"invalid-email","password":"password123"}`,
			expectCode: 400,
		},
		{
			name:       "short password",
			body:       `{"email":"test@example.com","password":"123"}`,
			expectCode: 400,
		},
		{
			name:       "empty JSON",
			body:       `{}`,
			expectCode: 400,
		},
		{
			name:       "invalid JSON",
			body:       `{invalid json}`,
			expectCode: 400,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			
			assert.Equal(t, tc.expectCode, w.Code)
			
			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "error")
		})
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	app := setupTestApp(t)
	
	// Register first user
	body := `{"email":"duplicate@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	
	// Try to register same email again
	req2 := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
	
	assert.Equal(t, 400, w2.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"].(string), "already registered")
}

func TestLoginEndpoint(t *testing.T) {
	app := setupTestApp(t)
	
	// First register a user
	regBody := `{"email":"login@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	
	// Now test login
	loginBody := `{"email":"login@example.com","password":"password123"}`
	req2 := httptest.NewRequest("POST", "/api/login", strings.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
	
	assert.Equal(t, 200, w2.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["token"])
	
	user, ok := resp["user"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "login@example.com", user["email"])
}

func TestLoginInvalidCredentials(t *testing.T) {
	app := setupTestApp(t)
	
	testCases := []struct {
		name string
		body string
	}{
		{
			name: "wrong password",
			body: `{"email":"nonexistent@example.com","password":"wrongpass"}`,
		},
		{
			name: "nonexistent user",
			body: `{"email":"nonexistent@example.com","password":"password123"}`,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			
			assert.Equal(t, 401, w.Code)
			
			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp["error"].(string), "invalid credentials")
		})
	}
}

func TestSendMessageEndpoint(t *testing.T) {
	app := setupTestApp(t)
	token := createTestUserAndGetToken(t, app)
	
	// Test sending valid message
	body := `{"message":"Hello support"}`
	req := httptest.NewRequest("POST", "/api/send", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 200, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "sent", resp["status"])
}

func TestSendMessageValidation(t *testing.T) {
	app := setupTestApp(t)
	token := createTestUserAndGetToken(t, app)
	
	testCases := []struct {
		name string
		body string
	}{
		{
			name: "empty message",
			body: `{"message":""}`,
		},
		{
			name: "missing message field",
			body: `{}`,
		},
		{
			name: "invalid JSON",
			body: `{invalid}`,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/send", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			
			assert.Equal(t, 400, w.Code)
			
			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "error")
		})
	}
}

func TestSendMessageUnauthorized(t *testing.T) {
	app := setupTestApp(t)
	
	body := `{"message":"Hello support"}`
	req := httptest.NewRequest("POST", "/api/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 401, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"].(string), "Invalid token")
}

func TestGetHistoryEndpoint(t *testing.T) {
	app := setupTestApp(t)
	token := createTestUserAndGetToken(t, app)
	
	// Add some messages first
	sendTestMessages(t, app, token, []string{"msg1", "msg2"})
	
	req := httptest.NewRequest("GET", "/api/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 200, w.Code)
	
	var resp map[string][]map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp["messages"], 2)
	
	// Check message content
	messages := resp["messages"]
	assert.Equal(t, "msg1", messages[0]["content"])
	assert.Equal(t, "msg2", messages[1]["content"])
	assert.Equal(t, "user", messages[0]["sender_type"])
	assert.Equal(t, "user", messages[1]["sender_type"])
}

func TestGetHistoryEmpty(t *testing.T) {
	app := setupTestApp(t)
	token := createTestUserAndGetToken(t, app)
	
	req := httptest.NewRequest("GET", "/api/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 200, w.Code)
	
	var resp map[string][]map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp["messages"], 0)
}

func TestGetHistoryUnauthorized(t *testing.T) {
	app := setupTestApp(t)
	
	req := httptest.NewRequest("GET", "/api/history", nil)
	// No Authorization header
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 401, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"].(string), "Invalid token")
}

func TestInvalidToken(t *testing.T) {
	app := setupTestApp(t)
	
	req := httptest.NewRequest("GET", "/api/history", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	assert.Equal(t, 401, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"].(string), "Invalid token")
}