package api

import (
	"log/slog"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type EditingHub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func NewEditingHub() *EditingHub {
	return &EditingHub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *EditingHub) AddClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = struct{}{}
}

func (h *EditingHub) RemoveClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
}

func (h *EditingHub) BroadcastMessage(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
			slog.Error("Failed to write message to client", "err", err)
			delete(h.clients, conn)
			conn.Close()
		}
	}
}

func (h *EditingHub) IsEmpty() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients) == 0
}

type HubManager struct {
	mu   sync.Mutex
	hubs map[string]*EditingHub
}

func NewHubManager() *HubManager {
	return &HubManager{hubs: make(map[string]*EditingHub)}
}

func (m *HubManager) GetHub(room string) *EditingHub {
	m.mu.Lock()
	defer m.mu.Unlock()
	if hub, exists := m.hubs[room]; exists {
		return hub
	}
	hub := NewEditingHub()
	m.hubs[room] = hub
	return hub
}

func (m *HubManager) CleanupHub(room string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if hub, exists := m.hubs[room]; exists {
		if hub.IsEmpty() {
			delete(m.hubs, room)
			slog.Info("Cleaning up empty hub", "room", room)
		} else {
			slog.Info("Hub not empty, not cleaning up", "room", room)
		}
	} else {
		slog.Warn("Attempted to clean up non-existent hub", "room", room)
	}
}
