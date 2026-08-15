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
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/handlers"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/httputil"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/logger"
	mw "github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/middleware"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/pubsub"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/redisclient"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/repository"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	// Root context cancels on SIGINT/SIGTERM. Every long-running subsystem
	// receives this ctx so they shut down together.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- Redis + pub/sub (publish side of the realtime path) ----

	redisClient, err := redisclient.New(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	bus := pubsub.NewRedis(redisClient)
	defer bus.Close()
	log.Info("redis and pubsub ready")

	// ---- Build the dependency graph: pool -> repos -> service -> handlers ----

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("db connected")

	usersRepo := repository.NewUsersRepository(pool)
	tokensRepo := repository.NewRefreshTokensRepository(pool)
	postsRepo := repository.NewPostsRepository(pool)
	commentsRepo := repository.NewCommentsRepository(pool)
	reactionsRepo := repository.NewReactionsRepository(pool)

	authService := services.NewAuthService(usersRepo, tokensRepo, services.AuthConfig{
		JWTSecret:  cfg.JWTSecret,
		AccessTTL:  cfg.JWTAccessTTL,
		RefreshTTL: cfg.JWTRefreshTTL,
	})
	feedService := services.NewFeedService(postsRepo, commentsRepo, reactionsRepo, bus, log)

	authHandler := handlers.NewAuthHandler(authService)
	feedHandler := handlers.NewFeedHandler(feedService)

	// ---- Router ----

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
	r.Use(mw.Auth(cfg.JWTSecret))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Mount("/auth", authHandler.Routes())

	r.Group(func(r chi.Router) {
		r.Use(mw.RequireAuth)
		r.Mount("/posts", feedHandler.Routes())
	})

	// ---- HTTP server with graceful shutdown ----

	srv := &http.Server{
		Addr:              cfg.DispatcherAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
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

	// Give in-flight requests a few seconds to finish before tearing down.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", "error", err)
	}

	log.Info("dispatcher stopped")
}
