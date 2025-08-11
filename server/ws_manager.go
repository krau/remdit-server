package server

import (
	"fmt"
	"log/slog"
	"remdit-server/config"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
)

type HubManager struct {
	mu   sync.Mutex
	hubs map[string]*EditingHub
}

func init() {
	go hubManager.startIntervalCleanup()
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
	slog.Info("cleaned up session", "sessionid", sessionID)
}

func (m *HubManager) ExistsHub(room string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.hubs[room]
	return exists
}

func (m *HubManager) startIntervalCleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupExpiredSessions()
	}
}

func (m *HubManager) cleanupExpiredSessions() {
	sessionTimeout := time.Duration(config.C.SessionTimeoutHours) * time.Hour
	m.mu.Lock()
	expiredSessions := make([]string, 0)
	now := time.Now()

	for sessionID, hub := range m.hubs {
		if hub.IsEmpty() && now.Sub(hub.lastActiveAt) > sessionTimeout {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}
	m.mu.Unlock()

	for _, sessionID := range expiredSessions {
		slog.Info("Cleaning up expired session", "sessionid", sessionID)
		m.CleanupSession(sessionID)
	}
}
