package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env identifies the deployment environment.
type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

// PubSubImpl selects the pub/sub backend implementation.
type PubSubImpl string

const (
	PubSubRedis   PubSubImpl = "redis"
	PubSubStreams PubSubImpl = "streams"
)

type Config struct {
	// ---- Shared ----
	AppEnv   Env
	LogLevel string

	// ---- Postgres ----
	PostgresDSN string

	// ---- Redis ----
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// ---- MinIO ----
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool

	// ---- Auth ----
	JWTSecret     []byte
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	// ---- Gateway ----
	GatewayID        string
	GatewayAddr      string
	GatewayPublicURL string

	// ---- Dispatcher ----
	DispatcherAddr string

	// ---- Presence ----
	PresenceHeartbeatInterval time.Duration
	PresenceTTL               time.Duration

	// ---- PubSub ----
	PubSub PubSubImpl
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:   Env(getEnv("APP_ENV", "dev")),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://app:app@localhost:5432/social?sslmode=disable"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "media"),

		JWTSecret: []byte(getEnv("JWT_SECRET", "")),

		GatewayID:        getEnv("GATEWAY_ID", "gateway-1"),
		GatewayAddr:      getEnv("GATEWAY_ADDR", ":8081"),
		GatewayPublicURL: getEnv("GATEWAY_PUBLIC_URL", "http://localhost:8081"),

		DispatcherAddr: getEnv("DISPATCHER_ADDR", ":8080"),

		PubSub: PubSubImpl(getEnv("PUBSUB_IMPL", "redis")),
	}

	var err error

	if cfg.RedisDB, err = strconv.Atoi(getEnv("REDIS_DB", "0")); err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	if cfg.MinIOUseSSL, err = strconv.ParseBool(getEnv("MINIO_USE_SSL", "false")); err != nil {
		return nil, fmt.Errorf("invalid MINIO_USE_SSL: %w", err)
	}
	if cfg.JWTAccessTTL, err = time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m")); err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}
	if cfg.JWTRefreshTTL, err = time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h")); err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}
	if cfg.PresenceHeartbeatInterval, err = time.ParseDuration(getEnv("PRESENCE_HEARTBEAT_INTERVAL", "30s")); err != nil {
		return nil, fmt.Errorf("invalid PRESENCE_HEARTBEAT_INTERVAL: %w", err)
	}
	if cfg.PresenceTTL, err = time.ParseDuration(getEnv("PRESENCE_TTL", "45s")); err != nil {
		return nil, fmt.Errorf("invalid PRESENCE_TTL: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func (c *Config) validate() error {
	if c.AppEnv != EnvDev && c.AppEnv != EnvProd {
		return fmt.Errorf("APP_ENV must be dev|prod, got %q", c.AppEnv)
	}

	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 bytes; set a strong secret")
	}

	if c.PubSub != PubSubRedis && c.PubSub != PubSubStreams {
		return fmt.Errorf("PUBSUB_IMPL must be redis|streams, got %q", c.PubSub)
	}

	if c.PresenceTTL <= c.PresenceHeartbeatInterval {
		return errors.New("PRESENCE_TTL must be greater than PRESENCE_HEARTBEAT_INTERVAL")
	}

	return nil
}

func (c *Config) Redacted() map[string]any {
	return map[string]any{
		"app_env":                     c.AppEnv,
		"log_level":                   c.LogLevel,
		"postgres_dsn":                redactDSN(c.PostgresDSN),
		"redis_addr":                  c.RedisAddr,
		"redis_db":                    c.RedisDB,
		"minio_endpoint":              c.MinIOEndpoint,
		"minio_bucket":                c.MinIOBucket,
		"minio_use_ssl":               c.MinIOUseSSL,
		"jwt_secret":                  "[REDACTED]",
		"jwt_access_ttl":              c.JWTAccessTTL.String(),
		"jwt_refresh_ttl":             c.JWTRefreshTTL.String(),
		"gateway_id":                  c.GatewayID,
		"gateway_addr":                c.GatewayAddr,
		"gateway_public_url":          c.GatewayPublicURL,
		"dispatcher_addr":             c.DispatcherAddr,
		"presence_heartbeat_interval": c.PresenceHeartbeatInterval.String(),
		"presence_ttl":                c.PresenceTTL.String(),
		"pubsub":                      c.PubSub,
	}
}

// redactDSN masks the password part of a DSN like:
// postgres://user:password@host:5432/db?sslmode=disable
// -> postgres://user:****@host:5432/db?sslmode=disable
//
// If the DSN does not match this shape, it is returned unchanged.
func redactDSN(dsn string) string {
	schemeIdx := strings.Index(dsn, "://")
	if schemeIdx == -1 {
		return dsn
	}

	credStart := schemeIdx + 3
	atIdx := strings.Index(dsn[credStart:], "@")
	if atIdx == -1 {
		return dsn
	}

	atIdx += credStart

	colonIdx := strings.Index(dsn[credStart:atIdx], ":")
	if colonIdx == -1 {
		return dsn
	}

	colonIdx += credStart

	return dsn[:colonIdx] + "***" + dsn[atIdx:]
}
