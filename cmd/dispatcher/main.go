package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/config"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/logger"
)

const serviceName = "dispatcher"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dispatcher: config load failed: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, cfg.LogLevel, serviceName)

	log.Info("starting",
		"addr", cfg.DispatcherAddr,
		"config", cfg.Redacted(),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	log.Info("shutdown signal received, stopping")
}
