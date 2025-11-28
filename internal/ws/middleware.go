package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// RateLimiter tracks message timestamps for rate limiting per connection
type RateLimiter struct {
	timestamps     []time.Time
	maxMessages    int
	windowDuration time.Duration
	mu             sync.Mutex
}

// NewRateLimiter creates a rate limiter allowing maxMessages per windowDuration
func NewRateLimiter(maxMessages int, windowDuration time.Duration) *RateLimiter {
	return &RateLimiter{
		timestamps:     make([]time.Time, 0),
		maxMessages:    maxMessages,
		windowDuration: windowDuration,
	}
}

// AllowMessage checks if a new message is allowed based on rate limit
func (rl *RateLimiter) AllowMessage() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowDuration)

	// Remove timestamps older than the window
	filtered := make([]time.Time, 0)
	for _, t := range rl.timestamps {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	rl.timestamps = filtered

	// Check if we've exceeded the limit
	if len(rl.timestamps) >= rl.maxMessages {
		return false
	}

	// Add current timestamp
	rl.timestamps = append(rl.timestamps, now)
	return true
}

// CommandHistory keeps track of the last N commands per connection
type CommandHistory struct {
	commands []string
	maxSize  int
	mu       sync.Mutex
}

// NewCommandHistory creates a new command history with specified size
func NewCommandHistory(size int) *CommandHistory {
	return &CommandHistory{
		commands: make([]string, 0, size),
		maxSize:  size,
	}
}

// Add adds a command to the history (circular buffer)
func (ch *CommandHistory) Add(command string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if len(ch.commands) >= ch.maxSize {
		ch.commands = ch.commands[1:] // Remove oldest
	}
	ch.commands = append(ch.commands, command)
}

// GetHistory returns all commands in history
func (ch *CommandHistory) GetHistory() []string {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Return a copy to avoid race conditions
	history := make([]string, len(ch.commands))
	copy(history, ch.commands)
	return history
}

// GetHistoryJSON returns history as JSON string
func (ch *CommandHistory) GetHistoryJSON() string {
	history := ch.GetHistory()
	data, err := json.Marshal(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
	if err != nil {
		return `{"error":"failed to marshal history"}`
	}
	return string(data)
}

// ClientHub manages all connected WebSocket clients for broadcasting
type ClientHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan BroadcastMessage
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

// BroadcastMessage contains the message and sender information
type BroadcastMessage struct {
	Payload []byte
	Sender  *websocket.Conn
}

// Global hub instance
var Hub *ClientHub

func init() {
	Hub = &ClientHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan BroadcastMessage, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
	go Hub.Run()
}

// Run starts the hub's main loop
func (h *ClientHub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			log.Printf("Client registered, total clients: %d", len(h.clients))

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				log.Printf("Client unregistered, total clients: %d", len(h.clients))
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Don't send back to sender (optional - can be changed)
				if client == msg.Sender {
					continue
				}

				// Try to write, if it fails, the connection will be cleaned up elsewhere
				err := client.WriteMessage(websocket.TextMessage, msg.Payload)
				if err != nil {
					log.Printf("error broadcasting to client: %v", err)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub
func (h *ClientHub) Register(conn *websocket.Conn) {
	h.register <- conn
}

// Unregister removes a client from the hub
func (h *ClientHub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

// Broadcast sends a message to all connected clients
func (h *ClientHub) Broadcast(payload []byte, sender *websocket.Conn) {
	h.broadcast <- BroadcastMessage{
		Payload: payload,
		Sender:  sender,
	}
}
