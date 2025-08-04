package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"remdit-server/config"
	"remdit-server/server"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "remdit-server",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger)
		config.InitConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.Serve(cmd.Context())
	},
}

func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		slog.Error("failed to execute root command", "err", err)
	}
}
