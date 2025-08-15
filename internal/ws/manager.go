package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Manager struct {
	clients map[int]*Client // userID -> client
	mu      sync.RWMutex
}

type Client struct {
	userID int
	conn   *websocket.Conn
	send   chan []byte
	manager *Manager
}

func NewManager() *Manager {
	return &Manager{
		clients: make(map[int]*Client),
	}
}

func (m *Manager) AddClient(userID int, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	client := &Client{
		userID:  userID,
		conn:    conn,
		send:    make(chan []byte, 256),
		manager: m,
	}
	
	m.clients[userID] = client
	go client.writePump()
	go client.readPump()
	
	// Send connection confirmation
	confirmMsg := map[string]string{
		"type": "connected",
	}
	data, _ := json.Marshal(confirmMsg)
	client.send <- data
}

func (m *Manager) RemoveClient(userID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if client, ok := m.clients[userID]; ok {
		close(client.send)
		client.conn.Close()
		delete(m.clients, userID)
	}
}

func (m *Manager) SendToUser(userID int, message []byte) {
	m.mu.RLock()
	client, ok := m.clients[userID]
	m.mu.RUnlock()
	
	if ok {
		select {
		case client.send <- message:
		default:
			// Client buffer full, close
			m.RemoveClient(userID)
		}
	}
}

func (m *Manager) GetClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

func (c *Client) readPump() {
	defer func() {
		c.manager.RemoveClient(c.userID)
		c.conn.Close()
	}()
	
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The manager closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			
			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}
			
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}