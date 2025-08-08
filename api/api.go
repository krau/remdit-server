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
	clients map[*websocket.Conn]struct{}
}

func newHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]struct{})}
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

func (h *Hub) isEmpty() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients) == 0
}

type HubManager struct {
	hubs map[string]*Hub
	mu   sync.Mutex
}

func (hm *HubManager) getHub(room string) *Hub {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hub, exists := hm.hubs[room]
	if !exists {
		hub = newHub()
		hm.hubs[room] = hub
	}
	return hub
}

func (hm *HubManager) cleanupHub(room string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if hub, exists := hm.hubs[room]; exists && hub.isEmpty() {
		delete(hm.hubs, room)
		slog.Info("Cleaned up empty hub", "room", room)
	}
}

func newHubManager() *HubManager {
	return &HubManager{hubs: make(map[string]*Hub)}
}

func Serve(ctx context.Context) {
	engine := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	engine.Use(cors.New(corsConfig))

	hubm := newHubManager()
	router := engine.Group("/api")
	router.GET("/ws/:room", func(ctx *gin.Context) {

		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		room := ctx.Param("room")
		hub := hubm.getHub(room)
		defer hubm.cleanupHub(room)
		hub.mu.Lock()
		hub.clients[conn] = struct{}{}
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
	router.POST("/file/:fileid", func(ctx *gin.Context) {
		// file id 即为 ws 中的 room
		fileID := ctx.Param("fileid")
		if fileID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "fileid is required"})
			return
		}
		var fileSaveReq FileSaveRequest
		if err := ctx.ShouldBindJSON(&fileSaveReq); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
			return
		}
		fileContent := fileSaveReq.Content
		// [TODO] 保存文件
		slog.Info("Saving file", "fileid", fileID, "content_length", len(fileContent))
		ctx.JSON(http.StatusOK, gin.H{"status": "success", "fileid": fileID, "content_length": len(fileContent)})
	})
	router.GET("/file/:fileid", func(ctx *gin.Context) {
		fileID := ctx.Param("fileid")
		if fileID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "fileid is required"})
			return
		}
		// [TODO] 获取文件内容
		slog.Info("Fetching file", "fileid", fileID)

		// 模拟

		fileContent := "This is a mock content for file " + fileID
		ctx.JSON(http.StatusOK, gin.H{"fileid": fileID, "content": fileContent})
	})

	serv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.C.APIHost, config.C.APIPort),
		Handler: engine,
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
