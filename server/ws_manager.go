package server

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

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
	return nil
}

func (m *HubManager) CreateHub(room string, sessionConn *websocket.Conn) (*EditingHub, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.hubs[room]; exists {
		return nil, fmt.Errorf("hub already exists for room: %s", room)
	}
	hub := NewEditingHub(room, sessionConn)
	m.hubs[room] = hub
	slog.Debug("Created new editing hub", "room", room)
	return hub, nil
}

// 当客户端ws断开时清理
func (m *HubManager) CleanupSession(sessionID string) {
	m.mu.Lock()
	hub, exists := m.hubs[sessionID]
	if !exists {
		m.mu.Unlock()
		return
	}
	delete(m.hubs, sessionID)
	m.mu.Unlock()

	hub.Cleanup()
	slog.Info("Completely cleaned up session", "sessionid", sessionID)
}

func (m *HubManager) ExistsHub(room string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.hubs[room]
	return exists
}
