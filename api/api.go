package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"remdit-server/config"
	"remdit-server/service"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/ratelimit"
)

var (
	wsUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	limiter ratelimit.Limiter
)

func leakBucket() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limiter.Take()
	}
}

func Serve(ctx context.Context, stor service.FileInfoStorage) {
	engine := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	engine.Use(cors.New(corsConfig))

	router := engine.Group("/api")
	limiter = ratelimit.New(max(config.C.APIRPS, 2))
	router.Use(leakBucket())

	hubManager := NewHubManager()
	router.GET("/socket/:room", func(ctx *gin.Context) {
		room := ctx.Param("room")
		if room == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "room is required"})
			return
		}
		fileid, err := uuid.Parse(room)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid room format"})
			return
		}
		fileInfo := stor.Get(ctx, fileid.String())
		if fileInfo == nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		slog.Info("WebSocket connection request", "room", room, "fileid", fileid)
		conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		hub := hubManager.GetHub(room)
		defer hubManager.CleanupHub(room)
		hub.AddClient(conn)
		defer hub.RemoveClient(conn)

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				slog.Error("Failed to read message", "err", err)
				break
			}
			if mt != websocket.BinaryMessage {
				continue
			}
			hub.BroadcastMessage(msg)
		}
	})
	router.PUT("/file/:fileid", func(ctx *gin.Context) {
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
		fileInfo := stor.Get(ctx, fileID)
		if fileInfo == nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		slog.Info("Saving file", "fileid", fileID, "content_length", len(fileSaveReq.Content))
		if err := service.WriteAndSyncFile(ctx, stor, fileID, []byte(fileSaveReq.Content)); err != nil {
			slog.Error("Failed to send file-save request", "err", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send file-save request"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "success", "fileid": fileID, "content_length": len(fileSaveReq.Content)})
	})
	router.GET("/file/:fileid", func(ctx *gin.Context) {
		fileID := ctx.Param("fileid")
		if fileID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "fileid is required"})
			return
		}
		fileInfo := stor.Get(ctx, fileID)
		if fileInfo == nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		slog.Info("Fetching file", "fileid", fileID)
		fileContent, err := os.ReadFile(fileInfo.Path())
		if err != nil {
			slog.Error("Failed to read file", "fileid", fileID, "err", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"fileid": fileID, "content": string(fileContent), "language": "plaintext"})
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
