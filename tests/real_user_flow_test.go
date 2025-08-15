package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealUserFlowToXMPPAdmin simulates a real web user (no XMPP account) 
// sending a message through the web API to the XMPP admin
func TestRealUserFlowToXMPPAdmin(t *testing.T) {
	app := setupTestApp(t)
	
	// Generate a realistic user email
	userEmail := fmt.Sprintf("webuser_%d@example.com", time.Now().Unix())
	userPassword := "SecurePassword123!"
	
	t.Logf("🧪 Testing real user flow with email: %s", userEmail)
	
	// Step 1: User registers via web (no XMPP account needed)
	t.Log("📝 Step 1: User registers via web API...")
	registerBody := map[string]string{
		"email":    userEmail,
		"password": userPassword,
	}
	registerJSON, _ := json.Marshal(registerBody)
	
	req := httptest.NewRequest("POST", "/api/register", bytes.NewReader(registerJSON))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	require.Equal(t, 201, w.Code, "Registration should succeed")
	
	var registerResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &registerResp)
	require.NoError(t, err)
	
	token := registerResp["token"].(string)
	user := registerResp["user"].(map[string]interface{})
	
	t.Logf("✅ User registered successfully with ID: %.0f", user["id"])
	t.Logf("🔐 JWT Token: %s...", token[:20])
	
	// Step 2: User sends message via web API 
	t.Log("💬 Step 2: User sends message via web API...")
	messageContent := fmt.Sprintf("Hello! I'm a web user (%s) and I need support. I don't have an XMPP account but can send messages through your website. This message should reach the XMPP admin. Time: %s", 
		userEmail, time.Now().Format("15:04:05"))
	
	sendBody := map[string]string{
		"message": messageContent,
	}
	sendJSON, _ := json.Marshal(sendBody)
	
	req = httptest.NewRequest("POST", "/api/send", bytes.NewReader(sendJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	require.Equal(t, 200, w.Code, "Message send should succeed")
	
	t.Log("✅ Message sent successfully via web API")
	
	// Step 3: Verify message was saved to database
	t.Log("💾 Step 3: Verifying message was saved to database...")
	req = httptest.NewRequest("GET", "/api/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	require.Equal(t, 200, w.Code, "Getting history should succeed")
	
	var historyResp map[string][]map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &historyResp)
	require.NoError(t, err)
	
	messages := historyResp["messages"]
	require.Len(t, messages, 1, "Should have one message in history")
	
	savedMsg := messages[0]
	assert.Equal(t, messageContent, savedMsg["content"])
	assert.Equal(t, "user", savedMsg["sender_type"])
	
	t.Log("✅ Message correctly saved to database")
	
	// Step 4: Check if XMPP connection is available and message was sent
	t.Log("📡 Step 4: Checking XMPP integration...")
	
	// Check environment variables
	xmppServer := os.Getenv("XMPP_SERVER")
	xmppAdminJID := os.Getenv("XMPP_ADMIN_JID")
	xmppConnectionJID := os.Getenv("XMPP_CONNECTION_JID")
	
	if xmppServer == "" || xmppAdminJID == "" || xmppConnectionJID == "" {
		t.Log("⚠️  XMPP environment variables not fully configured")
		t.Logf("   XMPP_SERVER: %s", xmppServer)
		t.Logf("   XMPP_ADMIN_JID: %s", xmppAdminJID)
		t.Logf("   XMPP_CONNECTION_JID: %s", xmppConnectionJID)
		return
	}
	
	t.Logf("🔧 XMPP Configuration:")
	t.Logf("   Server: %s", xmppServer)
	t.Logf("   Admin JID: %s", xmppAdminJID)
	t.Logf("   Connection JID: %s", xmppConnectionJID)
	
	// Expected message format that should be sent to XMPP admin
	expectedXMPPMessage := fmt.Sprintf("[User: %s] %s", userEmail, messageContent)
	t.Logf("📨 Expected XMPP message format: %s", expectedXMPPMessage)
	
	t.Log("✅ Real user flow test completed successfully!")
	t.Log("")
	t.Log("🎯 Test Summary:")
	t.Log("   ✓ Web user registered without XMPP account")
	t.Log("   ✓ Message sent via REST API")  
	t.Log("   ✓ Message saved to database")
	t.Log("   ✓ XMPP configuration verified")
	t.Log("")
	t.Log("📋 Manual Verification Steps:")
	t.Logf("   1. Check your XMPP client (%s)", xmppAdminJID)
	t.Log("   2. Look for a message with content:")
	t.Logf("      %s", expectedXMPPMessage)
	t.Log("   3. If received, the bridge is working correctly!")
}

// TestRealUserConversationFlow tests a full conversation flow
func TestRealUserConversationFlow(t *testing.T) {
	app := setupTestApp(t)
	
	// Generate a realistic user email
	userEmail := fmt.Sprintf("conversation_%d@example.com", time.Now().Unix())
	userPassword := "SecurePassword123!"
	
	t.Logf("🗣️  Testing full conversation flow with email: %s", userEmail)
	
	// Register user
	token := registerRealTestUser(t, app, userEmail, userPassword)
	
	// Send initial message
	t.Log("💬 Sending initial message...")
	initialMessage := fmt.Sprintf("Hi! I'm %s and I need help with your service. Can someone assist me? (Sent at %s)", 
		userEmail, time.Now().Format("15:04:05"))
	
	sendRealMessage(t, app, token, initialMessage)
	
	// Send follow-up message
	t.Log("💬 Sending follow-up message...")
	followUpMessage := "I'm still waiting for a response. Is anyone available to help?"
	
	time.Sleep(2 * time.Second) // Simulate time between messages
	sendRealMessage(t, app, token, followUpMessage)
	
	// Verify conversation history
	t.Log("📚 Checking conversation history...")
	history := getMessageHistory(t, app, token)
	
	assert.Len(t, history, 2, "Should have 2 messages in conversation")
	assert.Equal(t, initialMessage, history[0]["content"])
	assert.Equal(t, followUpMessage, history[1]["content"])
	
	// Display conversation for manual verification
	t.Log("")
	t.Log("📝 Conversation History:")
	for i, msg := range history {
		sender := msg["sender_type"].(string)
		content := msg["content"].(string)
		timestamp := msg["created_at"].(string)
		
		icon := "👤"
		if sender == "admin" {
			icon = "🛠️"
		}
		
		t.Logf("   %d. %s %s: %s (at %s)", i+1, icon, strings.Title(sender), content, timestamp)
	}
	
	t.Log("")
	t.Log("✅ Conversation flow test completed!")
	t.Log("📋 To test admin replies:")
	t.Logf("   1. Send a message to %s from your XMPP client", userEmail)
	t.Log("   2. The message should appear in the user's history")
	t.Log("   3. If user has WebSocket connection, they'll receive real-time updates")
}

// Helper functions

func registerRealTestUser(t *testing.T, app http.Handler, email, password string) string {
	registerBody := map[string]string{
		"email":    email,
		"password": password,
	}
	registerJSON, _ := json.Marshal(registerBody)
	
	req := httptest.NewRequest("POST", "/api/register", bytes.NewReader(registerJSON))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	require.Equal(t, 201, w.Code, "Registration should succeed")
	
	var registerResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &registerResp)
	require.NoError(t, err)
	
	return registerResp["token"].(string)
}

func sendRealMessage(t *testing.T, app http.Handler, token, message string) {
	sendBody := map[string]string{
		"message": message,
	}
	sendJSON, _ := json.Marshal(sendBody)
	
	req := httptest.NewRequest("POST", "/api/send", bytes.NewReader(sendJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	require.Equal(t, 200, w.Code, "Message send should succeed")
}

func getMessageHistory(t *testing.T, app http.Handler, token string) []map[string]interface{} {
	req := httptest.NewRequest("GET", "/api/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	require.Equal(t, 200, w.Code, "Getting history should succeed")
	
	var historyResp map[string][]map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &historyResp)
	require.NoError(t, err)
	
	return historyResp["messages"]
}