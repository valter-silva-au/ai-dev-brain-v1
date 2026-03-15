package server

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

// WSHub manages WebSocket connections and broadcasts HTML fragments
type WSHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
}

// NewWSHub creates a new WebSocket hub
func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*wsClient]struct{}),
	}
}

// HandleWS upgrades HTTP connections to WebSocket and registers clients
func (h *WSHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from any origin (local tool)
	})
	if err != nil {
		log.Printf("ws: accept error: %v", err)
		return
	}

	client := &wsClient{
		conn: conn,
		send: make(chan []byte, 64),
	}

	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	log.Printf("ws: client connected (%d total)", h.Count())

	// Writer goroutine — sends queued messages to the client
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
			conn.Close(websocket.StatusNormalClosure, "")
			log.Printf("ws: client disconnected (%d total)", h.Count())
		}()

		for msg := range client.send {
			err := conn.Write(context.Background(), websocket.MessageText, msg)
			if err != nil {
				return
			}
		}
	}()

	// Reader goroutine — reads messages from the client (keeps connection alive)
	for {
		_, _, err := conn.Read(context.Background())
		if err != nil {
			close(client.send)
			return
		}
	}
}

// Broadcast sends an HTML fragment to all connected clients
func (h *WSHub) Broadcast(html string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- []byte(html):
		default:
			// Client send buffer full — skip
		}
	}
}

// Count returns the number of connected clients
func (h *WSHub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
