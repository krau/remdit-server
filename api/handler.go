package api

import (
	"log/slog"
	"os"
	"path/filepath"
	"remdit-server/service/sshconn"
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
	client := hub.AddClientConn(conn)
	defer func() {
		client.Close()
		hubManager.CleanupHub(room)
	}()

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
		"fileid":     fileInfo.ID(),
		"content":    string(content),
		"roomexists": hubManager.ExistsHub(fileInfo.ID()),
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

var extToLang = map[string]string{
	"abap": "abap",

	"bat": "bat",
	"cmd": "bat",

	"bicep": "bicep",

	"c": "c",

	"mligo": "cameligo",

	"clj":  "clojure",
	"cljs": "clojure",
	"cljc": "clojure",

	"coffee": "coffeescript",
	"iced":   "coffeescript",

	"cpp": "cpp",
	"cc":  "cpp",
	"cxx": "cpp",
	"hpp": "cpp",
	"hh":  "cpp",

	"cs": "csharp",

	"css": "css",

	"dart": "dart",

	"Dockerfile": "dockerfile",
	"dockerfile": "dockerfile",

	"ecl": "ecl",

	"ex":  "elixir",
	"exs": "elixir",

	"fs":  "fsharp",
	"fsi": "fsharp",
	"fsx": "fsharp",

	"go": "go",

	"graphql": "graphql",
	"gql":     "graphql",

	"hbs":        "handlebars",
	"handlebars": "handlebars",

	"hcl": "hcl",
	"tf":  "hcl", // common for Terraform (HCL)

	"html": "html",
	"htm":  "html",

	"ini": "ini",

	"java": "java",

	"js":  "javascript",
	"mjs": "javascript",
	"cjs": "javascript",
	"jsx": "javascript",

	"json":  "json",
	"jsonc": "json",

	"jl": "julia",

	"kt":  "kotlin",
	"kts": "kotlin",

	"less": "less",

	"liquid": "liquid",

	"lua": "lua",

	"md":       "markdown",
	"markdown": "markdown",

	"pas": "pascal",
	"pp":  "pascal",

	"pl": "perl",
	"pm": "perl",

	"php":   "php",
	"phtml": "php",
	"inc":   "php",

	"txt": "plaintext",

	"ps1":  "powershell",
	"psm1": "powershell",
	"psd1": "powershell",

	"proto": "proto",

	"pug":  "pug",
	"jade": "pug",

	"py":  "python",
	"pyw": "python",
	"pyi": "python",

	"qs": "qsharp",

	"r":   "r",
	"rmd": "r",

	"cshtml": "razor",
	"vbhtml": "razor",

	"rst": "restructuredtext",

	"rb":   "ruby",
	"erb":  "ruby",
	"rake": "ruby",

	"rs": "rust",

	"scala": "scala",
	"sc":    "scala", // ammonite scripts

	"scm": "scheme",
	"ss":  "scheme",

	"scss": "scss",

	"sh":   "shell",
	"bash": "shell",

	"sol": "sol",

	"rq":     "sparql",
	"sparql": "sparql",

	"sql": "sql",

	"st": "st",

	"swift": "swift",

	"sv":  "systemverilog",
	"svh": "systemverilog",

	"tcl": "tcl",

	"twig": "twig",

	"ts":  "typescript",
	"tsx": "typescript",

	"vb": "vb",

	"v":  "verilog",
	"vh": "verilog",

	"xml": "xml",
	"xsd": "xml",
	"xsl": "xml",

	"yaml": "yaml",
	"yml":  "yaml",
}
