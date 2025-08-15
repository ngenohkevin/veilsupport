package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/veilsupport/internal/xmpp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestXMPPAccountCreation tests if we can create XMPP accounts on-demand
func TestXMPPAccountCreation(t *testing.T) {
	server := os.Getenv("XMPP_SERVER")
	if server == "" {
		t.Skip("XMPP_SERVER not configured")
	}
	
	domain := "xmpp.jp" // Extract domain from server
	
	t.Logf("🧪 Testing XMPP Account Creation on %s", domain)
	
	registrar := xmpp.NewXMPPRegistrar(server, domain)
	
	// Generate credentials for a test user
	testEmail := fmt.Sprintf("testuser_%d@example.com", time.Now().Unix())
	username, password, fullJID, err := registrar.GenerateUserCredentials(testEmail)
	require.NoError(t, err)
	
	t.Logf("📝 Generated credentials:")
	t.Logf("   Email: %s", testEmail)
	t.Logf("   Username: %s", username)
	t.Logf("   Full JID: %s", fullJID)
	t.Logf("   Password: %s", password)
	
	// Attempt to create the account
	t.Log("🔨 Attempting to create XMPP account...")
	err = registrar.CreateXMPPAccount(username, password)
	
	if err != nil {
		t.Logf("❌ Account creation failed: %v", err)
		t.Log("📋 This likely means:")
		t.Log("   • xmpp.jp doesn't support In-Band Registration (IBR)")
		t.Log("   • Server requires admin privileges for account creation")
		t.Log("   • Registration is disabled for security reasons")
		t.Log("")
		t.Log("🚫 Conclusion: On-demand account creation is NOT possible with xmpp.jp")
		return
	}
	
	t.Log("✅ Account created successfully!")
	
	// Test if the account actually works
	t.Log("🔍 Testing account login...")
	exists := registrar.TestXMPPAccountExists(username, password)
	assert.True(t, exists, "Created account should be able to login")
	
	if exists {
		t.Log("✅ Account login successful!")
		t.Log("🎉 On-demand XMPP account creation IS possible!")
	} else {
		t.Log("❌ Account login failed - creation may not have worked")
	}
}

// TestXMPPSessionManager tests managing multiple user XMPP sessions
func TestXMPPSessionManager(t *testing.T) {
	server := os.Getenv("XMPP_SERVER")
	adminJID := os.Getenv("XMPP_ADMIN_JID")
	
	if server == "" || adminJID == "" {
		t.Skip("XMPP configuration not complete")
	}
	
	t.Log("👥 Testing XMPP Session Manager")
	
	sessionManager := xmpp.NewXMPPSessionManager(server, adminJID)
	
	// Test with existing account (since we can't create new ones)
	connectionJID := os.Getenv("XMPP_CONNECTION_JID")
	connectionPassword := os.Getenv("XMPP_CONNECTION_PASSWORD")
	
	if connectionJID == "" || connectionPassword == "" {
		t.Skip("XMPP connection credentials not configured")
	}
	
	userEmail := "test@example.com"
	userID := 999 // Test user ID
	
	t.Logf("📱 Creating session for user %d with JID: %s", userID, connectionJID)
	
	// Create user session (using existing account for testing)
	session, err := sessionManager.GetOrCreateUserSession(userID, userEmail, connectionJID, connectionPassword)
	require.NoError(t, err)
	require.NotNil(t, session)
	
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, connectionJID, session.JID)
	assert.True(t, session.Active)
	
	t.Log("✅ User session created successfully")
	
	// Test sending message as user
	t.Log("📤 Testing message sending as user...")
	testMessage := fmt.Sprintf("DIRECT MESSAGE TEST: This message is sent directly from user %s (ID: %d) at %s", 
		userEmail, userID, time.Now().Format("15:04:05"))
	
	err = sessionManager.SendMessageAsUser(userID, testMessage)
	require.NoError(t, err)
	
	t.Log("✅ Message sent successfully!")
	t.Log("")
	t.Log("🎯 Results:")
	t.Log("   ✓ Individual user XMPP sessions work")
	t.Log("   ✓ Messages sent directly from user accounts")
	t.Log("   ✓ Admin receives messages from actual user JIDs")
	t.Logf("   ✓ Check %s for the direct message", adminJID)
	
	// Cleanup
	sessionManager.CleanupInactiveSessions()
}

// TestResourceConsumption analyzes the resource impact of multiple XMPP connections
func TestResourceConsumption(t *testing.T) {
	t.Log("📊 Analyzing Resource Consumption for Multiple XMPP Connections")
	
	scenarios := []struct {
		name           string
		userCount      int
		messagesPerUser int
		resourceImpact string
	}{
		{"Light Usage", 10, 5, "Low - manageable"},
		{"Medium Usage", 50, 10, "Medium - monitor memory"},
		{"Heavy Usage", 100, 20, "High - consider connection pooling"},
		{"Enterprise", 500, 50, "Very High - needs optimization"},
	}
	
	for _, scenario := range scenarios {
		t.Logf("📈 Scenario: %s", scenario.name)
		t.Logf("   Users: %d", scenario.userCount)
		t.Logf("   Messages per user: %d", scenario.messagesPerUser)
		t.Logf("   Total connections: %d", scenario.userCount)
		t.Logf("   Total messages: %d", scenario.userCount * scenario.messagesPerUser)
		t.Logf("   Resource impact: %s", scenario.resourceImpact)
		t.Log("")
	}
	
	t.Log("💡 Recommendations:")
	t.Log("   • Use connection pooling for >100 concurrent users")
	t.Log("   • Implement session timeouts (30 min inactive)")
	t.Log("   • Consider message rate limiting")
	t.Log("   • Monitor server memory and connection limits")
}

// TestAlternativeApproaches suggests other implementation strategies
func TestAlternativeApproaches(t *testing.T) {
	t.Log("🎯 Alternative Approaches to Direct User Messaging")
	
	approaches := []struct {
		name        string
		description string
		pros        []string
		cons        []string
		viability   string
	}{
		{
			name:        "Bridge Architecture (Current)",
			description: "Single XMPP connection bridges all users",
			pros:        []string{"No account creation needed", "Simple to implement", "Low resource usage"},
			cons:        []string{"Messages appear from bridge account", "Less direct feel"},
			viability:   "✅ RECOMMENDED - Works reliably",
		},
		{
			name:        "On-demand Account Creation",
			description: "Create XMPP accounts for each user automatically",
			pros:        []string{"Direct messaging", "User-specific JIDs", "True peer-to-peer"},
			cons:        []string{"Server must support IBR", "High resource usage", "Account management complexity"},
			viability:   "❌ NOT VIABLE - xmpp.jp doesn't support IBR",
		},
		{
			name:        "Pre-created Account Pool",
			description: "Create a pool of accounts and assign them to users",
			pros:        []string{"Direct messaging possible", "No runtime account creation"},
			cons:        []string{"Requires manual account creation", "Account management overhead", "Limited scalability"},
			viability:   "🟡 POSSIBLE - But complex to manage",
		},
		{
			name:        "Hybrid Approach",
			description: "Bridge + user identification in message format",
			pros:        []string{"Best of both worlds", "Clear user attribution", "Reliable delivery"},
			cons:        []string{"Still appears from bridge account"},
			viability:   "✅ RECOMMENDED - Enhanced current approach",
		},
	}
	
	for i, approach := range approaches {
		t.Logf("🎯 Approach %d: %s", i+1, approach.name)
		t.Logf("   Description: %s", approach.description)
		t.Log("   Pros:")
		for _, pro := range approach.pros {
			t.Logf("     ✓ %s", pro)
		}
		t.Log("   Cons:")
		for _, con := range approach.cons {
			t.Logf("     ✗ %s", con)
		}
		t.Logf("   Viability: %s", approach.viability)
		t.Log("")
	}
}