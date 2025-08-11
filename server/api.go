package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"remdit-server/config"
	"remdit-server/webembed"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var (
	hubManager = NewHubManager()
)

func Serve(ctx context.Context) {
	app := fiber.New(fiber.Config{
		JSONEncoder:             sonic.Marshal,
		JSONDecoder:             sonic.Unmarshal,
		EnableTrustedProxyCheck: true,
		TrustedProxies: []string{
			"localhost",
			"127.0.0.1",
		},
		ProxyHeader: fiber.HeaderXForwardedFor,
		BodyLimit:   10 * 1024 * 1024,
	})
	loggerCfg := logger.ConfigDefault
	loggerCfg.Format = "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${queryParams} | ${error}\n"
	app.Use(logger.New(loggerCfg))
	rg := app.Group("/api")
	rg.Use(limiter.New(limiter.Config{
		Max: max(config.C.APIRPM, 2),
	}))
	if config.C.APIKeyAuth && len(config.C.APIKeys) > 0 {
		rg.Use(keyauth.New(keyauth.Config{
			Next: func(c *fiber.Ctx) bool {
				if c.Path() == "/api/session" {
					return false
				}
				return true
			},
			KeyLookup: "header:X-API-Key",
			Validator: func(c *fiber.Ctx, s string) (bool, error) {
				hashedKey := sha256.Sum256([]byte(s))
				for _, key := range config.C.APIKeys {
					hashedApiKey := sha256.Sum256([]byte(key))
					if subtle.ConstantTimeCompare(hashedKey[:], hashedApiKey[:]) == 1 {
						return true, nil
					}
				}
				return false, keyauth.ErrMissingOrMalformedAPIKey
			},
		}))
	}
	rg.Post("/session", handleCreateSession)
	rg.Get("/session/:sessionid", handleSessionWSUpgrade)
	rg.Get("/session/:sessionid", websocket.New(handleSessionWSConn))
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
	if err := app.ShutdownWithTimeout(time.Second * 10); err != nil {
		slog.Error("Failed to gracefully shutdown API server", "err", err)
	} else {
		slog.Info("API server shutdown successfully")
	}
}
