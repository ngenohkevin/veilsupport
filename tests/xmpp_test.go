package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/veilsupport/internal/xmpp"
	"github.com/stretchr/testify/assert"
)

func setupTestXMPPClient(t *testing.T) *xmpp.XMPPClient {
	// Use XMPP credentials from .env file
	jid := os.Getenv("XMPP_CONNECTION_JID")
	password := os.Getenv("XMPP_CONNECTION_PASSWORD")
	server := os.Getenv("XMPP_SERVER")
	
	if jid == "" {
		jid = "test@localhost"
	}
	if password == "" {
		password = "testpass"
	}
	if server == "" {
		server = "localhost:5222"
	}
	
	client := xmpp.NewXMPPClient(jid, password, server)
	
	// Try to connect, skip test if XMPP server is not available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := client.ConnectWithContext(ctx)
	if err != nil {
		t.Skipf("XMPP server not available: %v", err)
	}
	
	return client
}

func TestXMPPConnection(t *testing.T) {
	// Use XMPP credentials from .env file
	jid := os.Getenv("XMPP_CONNECTION_JID")
	password := os.Getenv("XMPP_CONNECTION_PASSWORD")
	server := os.Getenv("XMPP_SERVER")
	
	if jid == "" {
		jid = "test@localhost"
	}
	if password == "" {
		password = "testpass"
	}
	if server == "" {
		server = "localhost:5222"
	}
	
	client := xmpp.NewXMPPClient(jid, password, server)
	
	// Test connection with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := client.ConnectWithContext(ctx)
	if err != nil {
		t.Skipf("XMPP server not available: %v", err)
		return
	}
	defer client.Close()
	
	assert.True(t, client.IsConnected())
}

func TestXMPPConnectionFailure(t *testing.T) {
	client := xmpp.NewXMPPClient("invalid@nonexistent", "wrongpass", "nonexistent:5222")
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err := client.ConnectWithContext(ctx)
	assert.Error(t, err)
	assert.False(t, client.IsConnected())
}

func TestSendXMPPMessage(t *testing.T) {
	client := setupTestXMPPClient(t)
	defer client.Close()
	
	// Test sending a message to admin
	adminJID := os.Getenv("XMPP_ADMIN_JID")
	if adminJID == "" {
		adminJID = "user_123@localhost"
	}
	err := client.SendMessage(adminJID, "Hello from test")
	assert.NoError(t, err)
	
	// Test sending message with empty body
	err = client.SendMessage("user_123@localhost", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message body cannot be empty")
	
	// Test sending message with invalid JID
	err = client.SendMessage("", "Hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recipient")
}

func TestReceiveXMPPMessage(t *testing.T) {
	client := setupTestXMPPClient(t)
	defer client.Close()
	
	// Create message channel
	messages := make(chan xmpp.XMPPMessage, 10)
	errorChan := make(chan error, 10)
	
	// Start listening in background
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	go func() {
		err := client.Listen(ctx, messages, errorChan)
		if err != nil && err != context.Canceled {
			errorChan <- err
		}
	}()
	
	// Give listener time to start
	time.Sleep(100 * time.Millisecond)
	
	// Simulate sending a message to ourselves (if supported by server)
	testMsg := "Hello test message"
	err := client.SendMessage(client.GetJID(), testMsg)
	if err != nil {
		t.Skipf("Cannot send message to self: %v", err)
		return
	}
	
	// Wait for message or timeout
	select {
	case msg := <-messages:
		assert.Equal(t, testMsg, msg.Body)
		assert.NotEmpty(t, msg.From)
	case err := <-errorChan:
		t.Fatalf("Error receiving message: %v", err)
	case <-ctx.Done():
		t.Skip("Timeout waiting for message - server may not support self-messaging")
	}
}

func TestXMPPMessageStructure(t *testing.T) {
	client := setupTestXMPPClient(t)
	defer client.Close()
	
	messages := make(chan xmpp.XMPPMessage, 1)
	errorChan := make(chan error, 1)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	go client.Listen(ctx, messages, errorChan)
	
	// Give listener time to start
	time.Sleep(100 * time.Millisecond)
	
	testMsg := "Structured message test"
	recipientJID := client.GetJID()
	
	err := client.SendMessage(recipientJID, testMsg)
	if err != nil {
		t.Skipf("Cannot send test message: %v", err)
		return
	}
	
	select {
	case msg := <-messages:
		assert.Equal(t, testMsg, msg.Body)
		assert.NotEmpty(t, msg.From)
		assert.Equal(t, recipientJID, msg.To)
	case err := <-errorChan:
		t.Fatalf("Error in message structure test: %v", err)
	case <-ctx.Done():
		t.Skip("Timeout - server may not support required features")
	}
}

func TestXMPPClientReconnection(t *testing.T) {
	// Use XMPP credentials from .env file
	jid := os.Getenv("XMPP_CONNECTION_JID")
	password := os.Getenv("XMPP_CONNECTION_PASSWORD")
	server := os.Getenv("XMPP_SERVER")
	
	if jid == "" {
		jid = "test@localhost"
	}
	if password == "" {
		password = "testpass"
	}
	if server == "" {
		server = "localhost:5222"
	}
	
	client := xmpp.NewXMPPClient(jid, password, server)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// First connection
	err := client.ConnectWithContext(ctx)
	if err != nil {
		t.Skipf("XMPP server not available: %v", err)
		return
	}
	
	assert.True(t, client.IsConnected())
	
	// Close connection
	client.Close()
	assert.False(t, client.IsConnected())
	
	// Reconnect
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	
	err = client.ConnectWithContext(ctx2)
	if err != nil {
		t.Fatalf("Failed to reconnect: %v", err)
	}
	
	assert.True(t, client.IsConnected())
	client.Close()
}

func TestXMPPConcurrentOperations(t *testing.T) {
	client := setupTestXMPPClient(t)
	defer client.Close()
	
	// Test sending multiple messages concurrently
	done := make(chan bool, 3)
	recipient := os.Getenv("XMPP_ADMIN_JID")
	if recipient == "" {
		recipient = "user_123@localhost"
	}
	
	for i := 0; i < 3; i++ {
		go func(id int) {
			err := client.SendMessage(recipient, fmt.Sprintf("Concurrent message %d", id))
			assert.NoError(t, err)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	timeout := time.After(5 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

