package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"remdit-server/config"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Serve(ctx context.Context) {
	router := gin.Default()
	corsConfig := cors.DefaultConfig()
	router.Use(cors.New(corsConfig))

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
