package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"remdit-server/config"
	"remdit-server/webembed"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var (
	hubManager = NewHubManager()
)

func Serve(ctx context.Context) {
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

	rg.Get("/socket/:room", handleRoomWSUpgrade)
	rg.Get("/socket/:room", websocket.New(handleRoomWSConn))

	rg.Use("/file/:fileid", handleFileMiddleware)
	rg.Put("/file/:fileid", handlePutFile)
	rg.Get("/file/:fileid", handleGetFile)

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
