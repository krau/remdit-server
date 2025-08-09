package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"remdit-server/config"
	"remdit-server/service"
	"remdit-server/webembed"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/google/uuid"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func Serve(ctx context.Context, stor service.FileInfoStorage) {
	app := fiber.New(fiber.Config{
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
		Prefork:     false,
	})
	loggerCfg := logger.ConfigDefault
	loggerCfg.Format = "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${queryParams} | ${error}\n"
	app.Use(logger.New(loggerCfg))
	rg := app.Group("/api")
	rg.Use(limiter.New(limiter.Config{
		Max: max(config.C.APIRPM, 2),
	}))

	hubManager := NewHubManager()

	rg.Get("/socket/:room", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			room := c.Params("room")
			if room == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "room is required"})
			}
			fileID, err := uuid.Parse(room)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid room format"})
			}
			fileInfo := stor.Get(c.Context(), fileID.String())
			if fileInfo == nil {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
			}
			slog.Info("WebSocket connection request", "room", room, "fileid", fileID)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	rg.Get("/socket/:room", websocket.New(func(c *websocket.Conn) {
		room := c.Params("room")
		hub := hubManager.GetHub(room)
		defer hubManager.CleanupHub(room)
		hub.AddClient(c)
		defer hub.RemoveClient(c)

		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				slog.Error("Failed to read message", "err", err)
				break
			}
			if mt != websocket.BinaryMessage {
				continue
			}
			hub.BroadcastMessage(msg)
		}
	}))
	rg.Use("/file/:fileid", func(c *fiber.Ctx) error {
		fileID := c.Params("fileid")
		if fileID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "fileid is required"})
		}
		if _, err := uuid.Parse(fileID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fileid format"})
		}
		fileInfo := stor.Get(c.Context(), fileID)
		if fileInfo == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		c.Locals("fileInfo", fileInfo)
		return c.Next()
	})
	rg.Put("/file/:fileid", func(c *fiber.Ctx) error {
		var fileSaveReq FileSaveRequest
		if err := c.BodyParser(&fileSaveReq); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "content is required"})
		}
		fileID := c.Params("fileid")
		fileInfo := c.Locals("fileInfo").(service.FileInfo)
		if fileInfo == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		slog.Info("Saving file", "fileid", fileID, "content_length", len(fileSaveReq.Content))
		if err := service.WriteAndSyncFile(c.Context(), stor, fileID, []byte(fileSaveReq.Content)); err != nil {
			slog.Error("Failed to write file", "fileid", fileID, "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "file saved successfully"})
	})
	rg.Get("/file/:fileid", func(c *fiber.Ctx) error {
		fileInfo := c.Locals("fileInfo").(service.FileInfo)
		if fileInfo == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		content, err := os.ReadFile(fileInfo.Path())
		if err != nil {
			slog.Error("Failed to read file", "fileid", fileInfo.ID(), "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"fileid":  fileInfo.ID(),
			"content": string(content),
			// "language": "plaintext", // [TODO]
		})
	})

	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(webembed.Static),
		NotFoundFile: "index.html", // let the frontend handle
	}))

	addr := fmt.Sprintf("%s:%d", config.C.APIHost, config.C.APIPort)
	go func() {
		if err := app.Listen(addr); err != nil {
			slog.Error("Failed to start API server", "err", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	slog.Info("API server is shutting down")
	if err := app.Shutdown(); err != nil {
		slog.Error("Failed to gracefully shutdown API server", "err", err)
	} else {
		slog.Info("API server shutdown successfully")
	}
}
