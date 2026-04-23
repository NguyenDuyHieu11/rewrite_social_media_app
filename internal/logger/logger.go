package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/config"
)

func New(env config.Env, level string, service string) *slog.Logger {
	return NewWithWriter(os.Stdout, env, level, service) // os.Stdout is a child of io.Writer
}

func NewWithWriter(w io.Writer, env config.Env, level string, service string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     parseLevel(level),
		AddSource: env == config.EnvDev,
	}

	var handler slog.Handler
	if env == config.EnvProd {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler).With("service", service)
}

func parseLevel(raw string) slog.Level {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info":
		fallthrough
	default:
		return slog.LevelInfo
	}
}
