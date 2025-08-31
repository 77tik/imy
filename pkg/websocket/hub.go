package websocket

import (
	"encoding/json"
	"sync"
	"time"
)

// Message represents a websocket message with routing information
type Message struct {
	Type      string      `json:"type"`           // "broadcast" or "direct"
	To        uint32      `json:"to,omitempty"`   // target user ID for direct messages
	From      uint32      `json:"from,omitempty"` // sender user ID
	Data      interface{} `json:"data"`           // message payload
	Timestamp int64       `json:"timestamp"`      // message timestamp
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Map of user ID to their active clients (supports multiple connections per user)
	userClients map[uint32]map[*Client]bool

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Inbound messages from the clients.
	broadcast chan []byte

	// Direct message channel
	direct chan *Message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:   make(chan []byte),
		direct:      make(chan *Message),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		userClients: make(map[uint32]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		case msg := <-h.direct:
			h.sendDirectMessage(msg)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	if client.userID != 0 {
		if h.userClients[client.userID] == nil {
			h.userClients[client.userID] = make(map[*Client]bool)
		}
		h.userClients[client.userID][client] = true
	}
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Remove from user clients map
		if client.userID != 0 {
			if userClients, exists := h.userClients[client.userID]; exists {
				delete(userClients, client)
				// If no more clients for this user, remove the user entry
				if len(userClients) == 0 {
					delete(h.userClients, client.userID)
				}
			}
		}
	}
}

// broadcastMessage sends message to all connected clients
func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			h.unregister <- client
		}
	}
}

// sendDirectMessage sends message to specific user
func (h *Hub) sendDirectMessage(msg *Message) {
	h.mu.RLock()
	userClients, exists := h.userClients[msg.To]
	// TODO:存储离线通知，等到下一次将所有离线通知一并发送过去
	if !exists {
		h.mu.RUnlock()
		return
	}

	clients := make([]*Client, 0, len(userClients))
	for client := range userClients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for _, client := range clients {
		select {
		case client.send <- messageBytes:
		default:
			h.unregister <- client
		}
	}
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID uint32, data interface{}) {
	msg := &Message{
		Type:      "direct",
		To:        userID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
	select {
	case h.direct <- msg:
	default:
		// Channel is full, message dropped
	}
}

// SendToUserFrom sends a message to a specific user with sender information
func (h *Hub) SendToUserFrom(fromUserID uint32, toUserID uint32, data interface{}) {
	msg := &Message{
		Type:      "direct",
		From:      fromUserID,
		To:        toUserID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
	select {
	case h.direct <- msg:
	default:
		// Channel is full, message dropped
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(data interface{}) {
	msg := &Message{
		Type:      "broadcast",
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- messageBytes:
	default:
		// Channel is full, message dropped
	}
}

// GetUserCount returns the number of connected users
func (h *Hub) GetUserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userClients)
}

// GetClientCount returns the total number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// IsUserOnline checks if a user is currently online
func (h *Hub) IsUserOnline(userID uint32) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.userClients[userID]
	return exists
}
