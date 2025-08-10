package api

import (
	"log/slog"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type WSEditingClient struct {
	conn *websocket.Conn
	send chan []byte
	hub  *EditingHub
	once sync.Once
}

func NewWSEditingClient(conn *websocket.Conn, hub *EditingHub) *WSEditingClient {
	c := &WSEditingClient{
		conn: conn,
		send: make(chan []byte, 16),
		hub:  hub,
	}
	go c.writePump()
	return c
}

func (c *WSEditingClient) writePump() {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
			slog.Error("client write error", "err", err)
			break
		}
	}
	c.conn.Close()
}

func (c *WSEditingClient) Close() {
	c.once.Do(func() {
		close(c.send)
		c.conn.Close()
		c.hub.RemoveClientConn(c)
	})
}

type EditingHub struct {
	mu      sync.Mutex
	clients map[*WSEditingClient]struct{}
}

func NewEditingHub() *EditingHub {
	return &EditingHub{clients: make(map[*WSEditingClient]struct{})}
}

func (h *EditingHub) AddClientConn(conn *websocket.Conn) *WSEditingClient {
	h.mu.Lock()
	defer h.mu.Unlock()
	cl := NewWSEditingClient(conn, h)
	h.clients[cl] = struct{}{}
	return cl
}

func (h *EditingHub) RemoveClientConn(c *WSEditingClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *EditingHub) BroadcastMessage(msg []byte) {
	h.mu.Lock()
	clients := make([]*WSEditingClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		select {
		case c.send <- msg:
			// ok
		default:
			// send channel 已满，认为客户端太慢或死掉，移除它
			slog.Warn("client send channel full, removing client")
			c.Close() // 触发移除
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
	hub, exists := m.hubs[room]
	if !exists {
		m.mu.Unlock()
		slog.Warn("Attempted to clean up non-existent hub", "room", room)
		return
	}
	m.mu.Unlock()

	if hub.IsEmpty() {
		m.mu.Lock()
		if h, ok := m.hubs[room]; ok && h.IsEmpty() {
			delete(m.hubs, room)
			slog.Debug("Cleaning up empty hub", "room", room)
		}
		m.mu.Unlock()
	}
}
