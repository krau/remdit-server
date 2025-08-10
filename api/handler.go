package api

import (
	"log/slog"
	"os"
	"remdit-server/service/sshconn"
	"remdit-server/service/stors/filestor"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func handleRoomWSUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		room := c.Params("room")
		if room == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "room is required"})
		}
		fileID, err := uuid.Parse(room)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid room format"})
		}
		fileInfo := filestor.Get(c.Context(), fileID.String())
		if fileInfo == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		slog.Info("WebSocket connection request", "room", room, "fileid", fileID)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func handleRoomWSConn(conn *websocket.Conn) {
	room := conn.Params("room")
	hub := hubManager.GetHub(room)
	client := hub.AddClientConn(conn)
	defer func() {
		client.Close()
		hubManager.CleanupHub(room)
	}()

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
}

func handleFileMiddleware(c *fiber.Ctx) error {
	fileID := c.Params("fileid")
	if fileID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "fileid is required"})
	}
	if _, err := uuid.Parse(fileID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid fileid format"})
	}
	fileInfo := filestor.Get(c.Context(), fileID)
	if fileInfo == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}
	c.Locals("fileInfo", fileInfo)
	return c.Next()
}

func handlePutFile(c *fiber.Ctx) error {
	var fileSaveReq FileSaveRequest
	if err := c.BodyParser(&fileSaveReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "content is required"})
	}
	fileID := c.Params("fileid")
	fileInfo := c.Locals("fileInfo").(filestor.FileInfo)
	if fileInfo == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}
	sshConn, ok := sshconn.Get(fileID)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "SSH connection not found"})
	}
	slog.Info("Saving file", "fileid", fileID, "content_length", len(fileSaveReq.Content))
	if err := filestor.WriteAndSyncFile(c.Context(), filestor.Default(), fileID, sshConn, []byte(fileSaveReq.Content)); err != nil {
		slog.Error("Failed to write file", "fileid", fileID, "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "file saved successfully"})
}

func handleGetFile(c *fiber.Ctx) error {
	fileInfo := c.Locals("fileInfo").(filestor.FileInfo)
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
}
