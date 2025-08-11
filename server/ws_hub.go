package server

import (
	"context"
	"fmt"
	"log/slog"
	"remdit-server/service/stors/filestor"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
)

type EditingHub struct {
	id             string
	clientsMu      sync.Mutex
	clients        map[*WSEditingClient]struct{} // 前端 ws 连接
	sessionConn    *websocket.Conn               // 客户端程序连接
	saveResultChan chan SaveResult
	chMu           sync.Mutex
}

func NewEditingHub(id string, sessionConn *websocket.Conn) *EditingHub {
	return &EditingHub{clients: make(map[*WSEditingClient]struct{}), id: id, sessionConn: sessionConn}
}

func (h *EditingHub) AddClientConn(conn *websocket.Conn) *WSEditingClient {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	cl := NewWSEditingClient(conn, h)
	h.clients[cl] = struct{}{}
	return cl
}

func (h *EditingHub) RemoveClientConn(c *WSEditingClient) {
	h.clientsMu.Lock()
	delete(h.clients, c)
	h.clientsMu.Unlock()
}

func (h *EditingHub) BroadcastMessage(msg []byte) {
	h.chMu.Lock()
	clients := make([]*WSEditingClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.chMu.Unlock()
	for _, c := range clients {
		go func(client *WSEditingClient) {
			select {
			case client.send <- msg:
			default:
				slog.Warn("Client send channel full, dropping message", "client", client.conn.RemoteAddr())
				client.Close()
			}
		}(c)
	}
}

func (h *EditingHub) NotifySessionSave(content string) error {
	if h.sessionConn == nil {
		return fmt.Errorf("no session connection available")
	}
	saveMsg := map[string]any{
		"type":    "save",
		"content": content,
	}
	return h.sessionConn.WriteJSON(saveMsg)
}

func (h *EditingHub) HandleSaveResult(success bool, reason string) {
	h.chMu.Lock()
	if h.saveResultChan == nil {
		h.saveResultChan = make(chan SaveResult, 1)
	}
	h.chMu.Unlock()
	select {
	case h.saveResultChan <- SaveResult{Success: success, Reason: reason}:
	default:
	}
}

func (h *EditingHub) WaitSaveResult() (bool, string, error) {
	h.chMu.Lock()
	if h.saveResultChan == nil {
		h.saveResultChan = make(chan SaveResult, 1)
	}
	h.chMu.Unlock()

	select {
	case result := <-h.saveResultChan:
		return result.Success, result.Reason, nil
	case <-time.After(10 * time.Second):
		return false, "timeout waiting for client response", fmt.Errorf("save confirmation timeout")
	}
}

func (h *EditingHub) Cleanup() {
	h.chMu.Lock()
	defer h.chMu.Unlock()

	h.clientsMu.Lock()
	clients := make([]*WSEditingClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.clients = make(map[*WSEditingClient]struct{})
	h.clientsMu.Unlock()

	for _, client := range clients {
		client.Close()
	}

	if h.sessionConn != nil {
		h.sessionConn.Close()
	}
	if err := filestor.Delete(context.Background(), h.id); err != nil {
		slog.Error("Failed to delete file", "fileid", h.id, "err", err)
	} else {
		slog.Info("Cleaned up session files", "fileid", h.id)
	}
}

func (h *EditingHub) IsEmpty() bool {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	return len(h.clients) == 0
}
