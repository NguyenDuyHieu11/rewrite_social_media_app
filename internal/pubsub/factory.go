package pubsub

import (
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/config"
	"github.com/redis/go-redis/v9"
)

// New returns the PubSub implementation selected by cfg.PubSub.
//
// Redis: client must be non-nil; the caller owns the client and must Close()
// it after closing the returned PubSub.
//
// Streams: client is ignored (stub does not use Redis).
func New(impl config.PubSubImpl, client *redis.Client) (PubSub, error) {
	switch impl {
	case config.PubSubRedis:
		if client == nil {
			return nil, fmt.Errorf("pubsub: redis implementation requires a non-nil *redis.Client")
		}
		return NewRedis(client), nil
	case config.PubSubStreams:
		return NewStreams(), nil
	default:
		return nil, fmt.Errorf("pubsub: unknown PUBSUB_IMPL %q", impl)
	}
}
