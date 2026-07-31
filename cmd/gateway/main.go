package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/config"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/feed"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/handlers"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/httputil"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/logger"
	mw "github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/middleware"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/pubsub"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/redisclient"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/subscription"

	"github.com/go-chi/chi/v5"
)

const serviceName = "gateway"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: config load failed: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, cfg.LogLevel, serviceName)
	log.Info("starting",
		"gateway_id", cfg.GatewayID,
		"addr", cfg.GatewayAddr,
		"config", cfg.Redacted(),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisClient, err := redisclient.New(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	bus := pubsub.NewRedis(redisClient)
	defer bus.Close()
	log.Info("redis and pubsub ready")

	// Home-feed realtime: in-memory fan-out + Redis bridge per post.
	// Hub refcount drives SUBSCRIBE/UNSUBSCRIBE; store holds per-browser channels.
	store := subscription.New()
	hub := feed.NewHub(bus, store)
	defer hub.Close()

	sseHandler := handlers.NewSSEHandler(hub, log)

	r := chi.NewRouter()
	r.Use(mw.RequestID)
	r.Use(mw.Logger(func(req *http.Request, status, bytes int, dur time.Duration) {
		log.Info("http",
			"method", req.Method,
			"path", req.URL.Path,
			"status", status,
			"bytes", bytes,
			"duration_ms", dur.Milliseconds(),
			"request_id", mw.RequestIDFromContext(req.Context()),
		)
	}))
	// QueryTokenAsBearer must precede Auth: EventSource clients can only pass
	// the JWT via ?access_token=, and Auth reads the Authorization header.
	r.Use(mw.QueryTokenAsBearer)
	r.Use(mw.Auth(cfg.JWTSecret))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth)
		r.Mount("/sse", sseHandler.Routes())
	})

	// SSE connections are long-lived; ReadTimeout/WriteTimeout must stay 0
	// or the server will cut active streams mid-event.
	srv := &http.Server{
		Addr:              cfg.GatewayAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received, stopping")
	case err := <-serverErr:
		log.Error("http server failed", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", "error", err)
	}

	log.Info("gateway stopped")
}
