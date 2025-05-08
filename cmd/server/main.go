package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Jonnydevp/subpub/internal/config"
	"github.com/Jonnydevp/subpub/internal/server"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	srv := server.New(cfg, logger)

	if err := srv.Start(); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		logger.Error("Failed to stop server", zap.Error(err))
	}
}
