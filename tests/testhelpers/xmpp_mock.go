package testhelpers

import (
	"context"
	"testing"
)

// MockXMPPClient provides a mock XMPP client for testing
type MockXMPPClient struct {
	SentMessages    []MockMessage
	ReceivedMessage chan MockMessage
}

type MockMessage struct {
	From    string
	To      string
	Content string
}

// NewMockXMPPClient creates a new mock XMPP client
func NewMockXMPPClient() *MockXMPPClient {
	return &MockXMPPClient{
		SentMessages:    make([]MockMessage, 0),
		ReceivedMessage: make(chan MockMessage, 100),
	}
}

// SendMessage simulates sending a message via XMPP
func (m *MockXMPPClient) SendMessage(from, to, content string) error {
	msg := MockMessage{
		From:    from,
		To:      to,
		Content: content,
	}
	m.SentMessages = append(m.SentMessages, msg)
	return nil
}

// WaitForMessage waits for a message to be received
func (m *MockXMPPClient) WaitForMessage(ctx context.Context) (MockMessage, error) {
	select {
	case msg := <-m.ReceivedMessage:
		return msg, nil
	case <-ctx.Done():
		return MockMessage{}, ctx.Err()
	}
}

// SetupTestXMPPServer sets up a mock XMPP server for integration tests
func SetupTestXMPPServer(t *testing.T) *MockXMPPServer {
	server := &MockXMPPServer{
		clients: make(map[string]*MockXMPPClient),
	}

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

type MockXMPPServer struct {
	clients map[string]*MockXMPPClient
}

func (s *MockXMPPServer) Close() {
	// Cleanup mock server resources
}

func (s *MockXMPPServer) ConnectionString() string {
	return "mock://localhost:5222"
}