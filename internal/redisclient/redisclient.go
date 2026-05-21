// Package redisclient builds a *redis.Client from configuration and verifies
// connectivity. One client is intended to live for the lifetime of the
// process; it is safe for concurrent use and pools its own connections for
// regular commands. Pub/Sub uses a separate dedicated connection that the
// internal/pubsub package manages on top of this client.
package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
	defaultPoolSize     = 10
	defaultMinIdleConns = 2
	defaultPingTimeout  = 2 * time.Second
)

// New constructs a *redis.Client and verifies it can reach the server with
// a Ping bounded by defaultPingTimeout. The caller owns the returned client
// and must Close() it during shutdown.
func New(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  defaultDialTimeout,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		PoolSize:     defaultPoolSize,
		MinIdleConns: defaultMinIdleConns,
	})

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis at %s: %w", addr, err)
	}

	return client, nil
}
