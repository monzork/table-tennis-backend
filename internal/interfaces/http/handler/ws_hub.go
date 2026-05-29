package handler

import (
	"encoding/json"
	"sync"

	fiberws "github.com/gofiber/websocket/v2"
)

// BracketHub manages WebSocket connections per tournament ID.
// Clients subscribe to a tournament and receive "reload" events when scores change.
type BracketHub struct {
	mu      sync.RWMutex
	clients map[string]map[*fiberws.Conn]bool // tournamentID → set of connections
}

var GlobalBracketHub = &BracketHub{
	clients: make(map[string]map[*fiberws.Conn]bool),
}

func (h *BracketHub) Register(tournamentID string, conn *fiberws.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[tournamentID] == nil {
		h.clients[tournamentID] = make(map[*fiberws.Conn]bool)
	}
	h.clients[tournamentID][conn] = true
}

func (h *BracketHub) Unregister(tournamentID string, conn *fiberws.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.clients[tournamentID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.clients, tournamentID)
		}
	}
}

// Broadcast sends a JSON message to all clients watching a specific tournament.
func (h *BracketHub) Broadcast(tournamentID string, event map[string]string) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	conns := h.clients[tournamentID]
	h.mu.RUnlock()

	for conn := range conns {
		conn.WriteMessage(1, payload) // 1 = TextMessage
	}
}

// WsBracketHandler is the Fiber handler for /ws/brackets/:tournamentId
func WsBracketHandler(c *fiberws.Conn) {
	tournamentID := c.Params("tournamentId")
	GlobalBracketHub.Register(tournamentID, c)
	defer GlobalBracketHub.Unregister(tournamentID, c)

	// Keep connection alive, reading any pings
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}
