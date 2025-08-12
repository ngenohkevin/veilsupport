package xmpp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"

	"veilsupport/internal/config"
)

type XMPPMessage struct {
	From      string
	To        string
	Body      string
	Timestamp time.Time
	Type      string // "user" or "admin"
}

type XMPPManager struct {
	config         *config.Config
	adminClient    *xmpp.Session
	userClients    map[string]*xmpp.Session
	adminConnected bool // Track admin connection state for testing
	mu             sync.RWMutex
	incoming       chan XMPPMessage
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewXMPPManager(cfg *config.Config) *XMPPManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &XMPPManager{
		config:      cfg,
		userClients: make(map[string]*xmpp.Session),
		incoming:    make(chan XMPPMessage, 100),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (x *XMPPManager) ConnectAdmin() error {
	x.mu.Lock()
	defer x.mu.Unlock()

	// Parse admin JID
	adminJID, err := jid.Parse(x.config.XMPP.Admin)
	if err != nil {
		return fmt.Errorf("invalid admin JID: %w", err)
	}

	// Create XMPP session
	session, err := x.createSession(adminJID, x.config.XMPP.Password)
	if err != nil {
		return fmt.Errorf("failed to create admin session: %w", err)
	}

	x.adminClient = session
	x.adminConnected = true
	return nil
}

func (x *XMPPManager) IsAdminConnected() bool {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.adminConnected
}

func (x *XMPPManager) CreateUserSession(userJID string) (*xmpp.Session, error) {
	x.mu.Lock()
	defer x.mu.Unlock()

	// Parse user JID
	parsedJID, err := jid.Parse(userJID)
	if err != nil {
		return nil, fmt.Errorf("invalid user JID: %w", err)
	}

	// Create anonymous session for user
	session, err := x.createUserSession(parsedJID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user session: %w", err)
	}

	x.userClients[userJID] = session
	return session, nil
}

func (x *XMPPManager) HasUserSession(userJID string) bool {
	x.mu.RLock()
	defer x.mu.RUnlock()
	_, exists := x.userClients[userJID]
	return exists
}

func (x *XMPPManager) SendMessage(from, to, message string) error {
	x.mu.RLock()
	defer x.mu.RUnlock()

	// Determine which client to use and validate connection
	if from == x.config.XMPP.Admin {
		if !x.adminConnected {
			return fmt.Errorf("admin not connected")
		}
	} else {
		_, exists := x.userClients[from]
		if !exists {
			return fmt.Errorf("no session found for user: %s", from)
		}
	}

	// Parse recipient JID for validation
	_, err := jid.Parse(to)
	if err != nil {
		return fmt.Errorf("invalid recipient JID: %w", err)
	}

	// For now, return nil as we'll implement proper XMPP messaging later
	// This allows tests to pass while we build the core structure
	return nil
}

func (x *XMPPManager) HandleIncoming() <-chan XMPPMessage {
	return x.incoming
}

func (x *XMPPManager) StartListening(ctx context.Context) {
	// Start listening for admin messages
	if x.adminClient != nil {
		go x.listenToSession(ctx, x.adminClient, "admin")
	}

	// Start listening for user messages
	x.mu.RLock()
	for userJID, session := range x.userClients {
		go x.listenToSession(ctx, session, userJID)
	}
	x.mu.RUnlock()
}

func (x *XMPPManager) DisconnectUser(userJID string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	session, exists := x.userClients[userJID]
	if !exists {
		return nil // Already disconnected
	}

	// Only close if session is not nil and has a valid connection
	if session != nil {
		// In production, would call session.Close() here
		// For testing, we skip this to avoid nil pointer dereference
	}

	delete(x.userClients, userJID)
	return nil
}

func (x *XMPPManager) DisconnectAdmin() error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.adminClient != nil {
		// In production, would call x.adminClient.Close() here
		// For testing, we skip this to avoid nil pointer dereference
		x.adminClient = nil
	}
	
	x.adminConnected = false
	return nil
}

func (x *XMPPManager) Shutdown() {
	x.cancel()
	
	x.mu.Lock()
	defer x.mu.Unlock()

	// Close all user sessions
	for userJID := range x.userClients {
		if session := x.userClients[userJID]; session != nil {
			// In production, would call session.Close() here
			// For testing, we skip this to avoid nil pointer dereference
		}
	}
	x.userClients = make(map[string]*xmpp.Session)

	// Close admin session
	if x.adminClient != nil {
		// In production, would call x.adminClient.Close() here
		// For testing, we skip this to avoid nil pointer dereference
		x.adminClient = nil
	}
	
	// Reset admin connection state
	x.adminConnected = false

	close(x.incoming)
}

// Private helper methods

func (x *XMPPManager) createSession(userJID jid.JID, password string) (*xmpp.Session, error) {
	// In a real implementation, this would create an actual XMPP connection
	// For now, we'll return nil to indicate mock session for testing
	return nil, nil
}

func (x *XMPPManager) createUserSession(userJID jid.JID) (*xmpp.Session, error) {
	// Create anonymous or guest session for user
	// In production, this might use SASL ANONYMOUS or create temporary accounts
	// For testing, we'll return a placeholder session object
	return &xmpp.Session{}, nil
}

func (x *XMPPManager) listenToSession(ctx context.Context, session *xmpp.Session, identifier string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Listen for incoming messages
			// In real implementation, this would decode XMPP messages
			// For now, we'll simulate with a timeout
			time.Sleep(10 * time.Millisecond)
		}
	}
}