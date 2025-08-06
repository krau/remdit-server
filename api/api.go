package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"remdit-server/config"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func newHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

func (h *Hub) broadcast(sender *websocket.Conn, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if c != sender {
			c.WriteMessage(websocket.BinaryMessage, msg)
		}
	}
}

func Serve(ctx context.Context) {
	router := gin.Default()
	hub := newHub()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	router.Use(cors.New(corsConfig))

	router.GET("/ws/:room", func(ctx *gin.Context) {
		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		hub.mu.Lock()
		hub.clients[conn] = true
		hub.mu.Unlock()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if mt != websocket.BinaryMessage {
				continue
			}
			hub.broadcast(conn, msg)
		}

		hub.mu.Lock()
		delete(hub.clients, conn)
		hub.mu.Unlock()
	})

	serv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.C.APIHost, config.C.APIPort),
		Handler: router,
	}
	go func() {
		<-ctx.Done()
		slog.Info("API server is shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := serv.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shutdown server", "err", err)
			return
		}
		slog.Info("API server stopped")
	}()
	if err := serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Failed to start API server", "err", err)
		return
	}
}
