package server

import (
	"log/slog"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

// 前端ws连接客户端
type WSEditingClient struct {
	conn *websocket.Conn
	send chan []byte
	hub  *EditingHub
	once sync.Once
}

func NewWSEditingClient(conn *websocket.Conn, hub *EditingHub) *WSEditingClient {
	c := &WSEditingClient{
		conn: conn,
		send: make(chan []byte, 64),
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
}

func (c *WSEditingClient) Close() {
	c.once.Do(func() {
		close(c.send)
		c.conn.Close()
		c.hub.RemoveClientConn(c)
	})
}
