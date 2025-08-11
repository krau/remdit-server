package server

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"remdit-server/config"
	"remdit-server/service/stors/filestor"
	"strings"

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
	if hub == nil || hub.sessionConn == nil {
		slog.Error("No editing hub found for room", "room", room)
		conn.Close()
		return
	}

	client := hub.AddClientConn(conn)
	defer client.Close()

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Info("WebSocket connection closed", "room", room)
				return
			}
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
	fileInfo := c.Locals("fileInfo").(filestor.File)
	if fileInfo == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}

	hub := hubManager.GetHub(fileID)
	if hub == nil || hub.sessionConn == nil {
		slog.Error("No editing hub found for file", "fileid", fileID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "editing hub not found"})
	}

	slog.Info("Saving file", "fileid", fileID, "content_length", len(fileSaveReq.Content))
	// 保存文件到本地
	if err := os.WriteFile(fileInfo.Path(), []byte(fileSaveReq.Content), 0644); err != nil {
		slog.Error("Failed to write file", "fileid", fileID, "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}

	// 通知客户端保存文件
	if err := hub.NotifySessionSave(fileSaveReq.Content); err != nil {
		slog.Warn("Failed to notify session about file save", "fileid", fileID, "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to notify client"})
	}

	// 等待客户端确认保存结果
	success, reason, err := hub.WaitSaveResult()
	if err != nil {
		slog.Error("Failed to get save confirmation from client", "fileid", fileID, "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save confirmation failed", "reason": err.Error()})
	}

	if !success {
		slog.Error("Client reported save failure", "fileid", fileID, "reason", reason)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "client save failed", "reason": reason})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "file saved successfully"})
}

func handleGetFile(c *fiber.Ctx) error {
	fileInfo := c.Locals("fileInfo").(filestor.File)
	if fileInfo == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}
	content, err := os.ReadFile(fileInfo.Path())
	if err != nil {
		slog.Error("Failed to read file", "fileid", fileInfo.ID(), "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"fileid":     fileInfo.ID(),
		"content":    string(content),
		"roomexists": hubManager.ExistsHub(fileInfo.ID()),
		"filename":   fileInfo.Name(),
		"language": func() string {
			ext := filepath.Ext(fileInfo.Name())
			if ext == "" {
				return "plaintext"
			}
			ext = strings.TrimPrefix(ext, ".")
			if lang, ok := extToLang[strings.ToLower(ext)]; ok {
				return lang
			}
			return "plaintext"
		}(),
	})
}

func handleSessionWSUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		sessionid := c.Params("sessionid")
		if sessionid == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "sessionid is required"})
		}
		fileID, err := uuid.Parse(sessionid)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sessionid format"})
		}
		fileInfo := filestor.Get(c.Context(), fileID.String())
		if fileInfo == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		c.Locals("fileInfo", fileInfo)
		slog.Info("WebSocket connection request", "sessionid", sessionid, "fileid", fileID)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func handleSessionWSConn(conn *websocket.Conn) {
	fileInfo := conn.Locals("fileInfo").(filestor.File)
	if fileInfo == nil {
		slog.Error("File info not found in session WS connection")
		conn.Close()
		return
	}

	sessionID := fileInfo.ID()
	hub, err := hubManager.CreateHub(sessionID, conn)
	if err != nil {
		slog.Error("Failed to create editing hub for session", "sessionid", sessionID, "err", err)
		conn.Close()
		return
	}

	slog.Info("Session WebSocket connected", "sessionid", sessionID)

	defer func() {
		hubManager.CleanupSession(sessionID)
		slog.Info("Session WebSocket disconnected and cleaned up", "sessionid", sessionID)
	}()

	for {
		var msg map[string]any
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Info("Session WebSocket connection closed", "sessionid", sessionID)
				return
			}
			slog.Error("Failed to read session message", "err", err)
			break
		}
		// 处理客户端返回的保存结果
		if msgType, ok := msg["type"].(string); ok && msgType == "save_result" {
			success, _ := msg["success"].(bool)
			reason, _ := msg["reason"].(string)

			// 通知Hub处理保存结果
			hub.HandleSaveResult(success, reason)

			if success {
				slog.Info("Client confirmed file save success", "sessionid", sessionID)
			} else {
				slog.Error("Client reported file save failure", "sessionid", sessionID, "reason", reason)
			}
		}
	}
}

func handleCreateSession(c *fiber.Ctx) error {
	file, err := c.FormFile("document")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	if file.Size > config.MaxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file size exceeds limit"})
	}
	fileID := uuid.New().String()
	filePath := filepath.Join(config.C.UploadsDir, fileID, file.Filename)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create directory"})
	}

	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}
	slog.Info("File uploaded", "fileid", fileID, "filename", file.Filename, "size", file.Size)
	if err := filestor.Save(c.Context(), fileID, filestor.NewFileInfo(fileID, filePath, file.Filename)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file info"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"sessionid": fileID,
		"editurl":   fmt.Sprintf("%s/edit/%s", config.C.ServerURLs[rand.Intn(len(config.C.ServerURLs))], fileID),
	})
}
