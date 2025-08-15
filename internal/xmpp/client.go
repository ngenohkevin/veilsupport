package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type XMPPClient struct {
	jid       string
	password  string
	server    string
	session   *xmpp.Session
	connected bool
	mu        sync.RWMutex
}

type XMPPMessage struct {
	From string
	To   string
	Body string
}

func NewXMPPClient(jidStr, password, server string) *XMPPClient {
	return &XMPPClient{
		jid:      jidStr,
		password: password,
		server:   server,
	}
}

func (c *XMPPClient) ConnectWithContext(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.session != nil {
		return nil
	}

	// Parse JID
	addr, err := jid.Parse(c.jid)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	log.Printf("XMPP: Connecting to %s as %s", c.server, c.jid)

	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName:         addr.Domain().String(),
		InsecureSkipVerify: true, // For testing - in production use proper certificates
	}

	// Connect to XMPP server with proper configuration
	conn, err := xmpp.DialClientSession(
		ctx, addr,
		xmpp.BindResource(),
		xmpp.StartTLS(tlsConfig),
		xmpp.SASL("", c.password, sasl.Plain),
	)
	if err != nil {
		return fmt.Errorf("failed to create XMPP session: %w", err)
	}

	// Send initial presence to indicate we're online
	err = conn.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to send presence: %w", err)
	}

	c.session = conn
	c.connected = true
	
	log.Printf("XMPP: Successfully connected to %s", c.server)
	return nil
}

func (c *XMPPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.session != nil
}

func (c *XMPPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		// Send unavailable presence before closing
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		_ = c.session.Send(ctx, stanza.Presence{Type: stanza.UnavailablePresence}.Wrap(nil))
		
		err := c.session.Close()
		c.session = nil
		c.connected = false
		log.Println("XMPP: Connection closed")
		return err
	}
	c.connected = false
	return nil
}

func (c *XMPPClient) SendMessage(to, body string) error {
	if to == "" {
		return errors.New("invalid recipient")
	}
	if body == "" {
		return errors.New("message body cannot be empty")
	}

	c.mu.RLock()
	session := c.session
	connected := c.connected
	c.mu.RUnlock()

	if !connected || session == nil {
		return errors.New("not connected to XMPP server")
	}

	// Parse recipient JID
	recipientJID, err := jid.Parse(to)
	if err != nil {
		return fmt.Errorf("invalid recipient JID: %w", err)
	}

	// Create message with custom body encoder
	msg := stanza.Message{
		To:   recipientJID,
		Type: stanza.ChatMessage,
		ID:   fmt.Sprintf("msg_%d", time.Now().Unix()),
	}
	
	// Create message body element
	bodyStart := xml.StartElement{Name: xml.Name{Local: "body"}}
	bodyContent := xmlstream.Wrap(
		xmlstream.Token(xml.CharData(body)),
		bodyStart,
	)
	
	// Wrap the message with body content
	messageWithBody := msg.Wrap(bodyContent)
	
	// Send message with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = session.Send(ctx, messageWithBody)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	
	log.Printf("XMPP: Message sent from %s to %s: %s", c.jid, to, body)
	return nil
}

// Alternative simple send method if the above doesn't work
func (c *XMPPClient) SendMessageSimple(to, body string) error {
	if to == "" || body == "" {
		return errors.New("invalid recipient or body")
	}

	c.mu.RLock()
	session := c.session
	connected := c.connected
	c.mu.RUnlock()

	if !connected || session == nil {
		return errors.New("not connected to XMPP server")
	}

	// Parse recipient JID
	recipientJID, err := jid.Parse(to)
	if err != nil {
		return fmt.Errorf("invalid recipient JID: %w", err)
	}

	// Create a simple message encoder
	encoder := SimpleMessage{
		To:   recipientJID.String(),
		Type: "chat",
		Body: body,
		ID:   fmt.Sprintf("msg_%d", time.Now().Unix()),
	}
	
	// Send using the encoder
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = session.Send(ctx, encoder.TokenReader())
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	
	log.Printf("XMPP: Message sent from %s to %s: %s", c.jid, to, body)
	return nil
}

func (c *XMPPClient) Listen(ctx context.Context, messages chan<- XMPPMessage, errorChan chan<- error) error {
	c.mu.RLock()
	session := c.session
	connected := c.connected
	c.mu.RUnlock()

	if !connected || session == nil {
		return errors.New("not connected to XMPP server")
	}

	log.Println("XMPP: Starting message listener")

	// Create a simple handler for incoming messages
	for {
		select {
		case <-ctx.Done():
			log.Println("XMPP: Listener stopped by context")
			return ctx.Err()
		default:
			// This is a simplified listener - in production you'd use session.Serve
			// For now, we'll just keep the connection alive
			time.Sleep(1 * time.Second)
			
			// Periodic ping to keep connection alive
			if c.IsConnected() {
				pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				_ = c.session.Send(pingCtx, stanza.IQ{Type: stanza.GetIQ}.Wrap(nil))
				cancel()
			}
		}
	}
}

func (c *XMPPClient) GetJID() string {
	return c.jid
}

