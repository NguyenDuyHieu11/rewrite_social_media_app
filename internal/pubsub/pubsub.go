package pubsub

import (
	"context"
	"errors"
)

// Message is one event delivered from the bus.
type Message struct {
	Channel string // e.g. "post:42"
	Payload []byte // opaque; pubsub does not interpret
}

// Sentinel errors.
var (
	// ErrClosed is returned by Publish/Subscribe/Unsubscribe after Close.
	ErrClosed = errors.New("pubsub: closed")

	// ErrNotImplemented is returned by stub backends (e.g. Streams pre-M-later).
	ErrNotImplemented = errors.New("pubsub: not implemented")
)

// PubSub is the abstraction the gateway and dispatcher use to move events
// between processes. Implementations: RedisPubSub (live), StreamsPubSub (stub).
//
// All methods are safe for concurrent use.
type PubSub interface {
	// Publish broadcasts payload to every subscriber of channel.
	// Fire-and-forget at the protocol level: if no one is subscribed,
	// the message is discarded by the broker.
	Publish(ctx context.Context, channel string, payload []byte) error

	// Subscribe registers interest in a channel and returns a receive-only
	// channel of Messages. Each call returns a *distinct* Go channel, even
	// for the same Redis channel name; multiple callers on the same gateway
	// are deduplicated internally by the implementation (one Redis SUBSCRIBE,
	// many Go consumers).
	//
	// The returned channel is closed when:
	//   - Unsubscribe is called for this (channel, returned-chan) pair, or
	//   - Close is called on the PubSub, or
	//   - the underlying transport drops irrecoverably.
	Subscribe(ctx context.Context, channel string) (<-chan Message, error)

	// Unsubscribe releases a previous Subscribe. The exact channel value
	// returned by Subscribe must be passed back (this is how we identify
	// which subscription to drop when many goroutines share one Redis channel).
	Unsubscribe(ctx context.Context, channel string, sub <-chan Message) error

	// Close shuts the bus down and releases all transport resources.
	// Idempotent. Subsequent calls return nil.
	Close() error
}
