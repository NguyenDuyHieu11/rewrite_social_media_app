// Package feed implements the gateway-side home-feed event bridge.
//
// # Problem
//
// Browsers open an SSE connection per post they are viewing. The dispatcher
// publishes events to Redis channel post:{id}. Each gateway must:
//  1. SUBSCRIBE to Redis only while at least one local browser cares about that post.
//  2. Fan Redis messages into per-connection Go channels (subscription.Store).
//  3. UNSUBSCRIBE and stop the bridge when the last local browser leaves.
//
// # Why Hub exists
//
// subscription.Store tracks per-browser channels but does not talk to Redis.
// pubsub.PubSub talks to Redis but knows nothing about HTTP connections.
// Hub sits between them and owns the *per-post* lifecycle on this gateway:
//
//	Redis post:{id}  ──bridge──►  subscription.Store  ──►  SSE handler(s)
//
// Without Hub, tying the Redis bridge to a single handler's request context
// would tear down Redis when the first browser disconnects while others on
// the same gateway are still watching.
//
// # Refcounting
//
// Hub maintains entry.refs = number of active Acquire calls for that postID.
// The store's firstForPost/lastForPost flags are ignored here; Hub drives
// Redis subscribe/unsubscribe from entry.refs alone. The store still tracks
// individual connection IDs for fan-out and Unsubscribe.
//
// # Concurrency
//
// All map mutations and refcount changes run under Hub.mu. The bridge goroutine
// only calls store.Publish (which uses its own RWMutex). Acquire/Release must
// be paired per SSE connection; the handler defers Release on exit.
//
// # Shutdown
//
// Call Close once during gateway shutdown. It cancels all bridges, unsubscribes
// Redis, and closes the subscription store.
package feed

import (
	"context"
	"sync"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/channels"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/pubsub"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/subscription"
)

// Hub bridges Redis pub/sub and the in-memory subscription store for one
// gateway instance. Safe for concurrent use.
type Hub struct {
	mu    sync.Mutex
	bus   pubsub.PubSub           // upstream: Redis (or test double)
	store *subscription.Store     // downstream: per-browser delivery channels
	posts map[string]*postEntry   // postID -> active bridge state
}

// postEntry holds Redis subscription state for one post on this gateway.
type postEntry struct {
	refs   int                      // open SSE connections for this post
	redis  <-chan pubsub.Message    // receive-only; identity for bus.Unsubscribe
	cancel context.CancelFunc       // stops the bridge goroutine
}

// NewHub wires a PubSub bus and subscription store. Neither may be nil.
// The caller must call Close on shutdown.
func NewHub(bus pubsub.PubSub, store *subscription.Store) *Hub {
	return &Hub{
		bus:   bus,
		store: store,
		posts: make(map[string]*postEntry),
	}
}

// Acquire registers one browser's interest in postID for userID.
//
// On the first Acquire for a given postID on this gateway, Hub subscribes to
// channels.PostChannel(postID) and starts a bridge goroutine that copies Redis
// payloads into store.Publish. Subsequent Acquire calls for the same post only
// increment refs and add another entry in the store.
//
// Returns:
//   - local — read-only channel of opaque events ([]byte) for the SSE handler
//   - connID — opaque handle required by Release / store.Unsubscribe
//   - err — if Redis Subscribe fails (e.g. bus closed, not connected)
//
// ctx is reserved for future cancellation during Acquire; the Redis bridge
// uses a detached context so it outlives any single HTTP request.
func (h *Hub) Acquire(ctx context.Context, postID, userID string) (<-chan subscription.Event, string, error) {
	_ = ctx
	ch := make(chan subscription.Event, 64)

	h.mu.Lock()
	defer h.mu.Unlock()

	entry := h.posts[postID]
	if entry == nil {
		bridgeCtx, cancel := context.WithCancel(context.Background())
		redisCh, err := h.bus.Subscribe(bridgeCtx, channels.PostChannel(postID))
		if err != nil {
			cancel()
			return nil, "", err
		}
		entry = &postEntry{refs: 0, redis: redisCh, cancel: cancel}
		h.posts[postID] = entry
		go h.bridge(bridgeCtx, postID, redisCh)
	}
	entry.refs++

	connID, _ := h.store.Subscribe(postID, userID, ch)
	return ch, connID, nil
}

// Release undoes one Acquire for the connection identified by connID.
//
// Decrements the per-post refcount. When refs reaches zero, the bridge
// goroutine is cancelled, Redis is unsubscribed, and the postEntry is removed.
// The store entry for connID is always removed regardless of refcount.
//
// Safe to call if connID is unknown (store no-op). ctx is passed to
// bus.Unsubscribe for Redis teardown.
func (h *Hub) Release(ctx context.Context, postID, userID, connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.store.Unsubscribe(postID, userID, connID)

	entry := h.posts[postID]
	if entry == nil {
		return
	}
	entry.refs--
	if entry.refs > 0 {
		return
	}

	entry.cancel()
	_ = h.bus.Unsubscribe(ctx, channels.PostChannel(postID), entry.redis)
	delete(h.posts, postID)
}

// bridge copies messages from Redis into the in-memory store until ctx is
// cancelled or redisCh closes.
func (h *Hub) bridge(ctx context.Context, postID string, redisCh <-chan pubsub.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-redisCh:
			if !ok {
				return
			}
			h.store.Publish(postID, msg.Payload)
		}
	}
}

// Close shuts down all active post bridges and the subscription store.
// Idempotent from the caller's perspective if invoked once at process exit.
// Do not call Acquire after Close.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for postID, entry := range h.posts {
		entry.cancel()
		_ = h.bus.Unsubscribe(context.Background(), channels.PostChannel(postID), entry.redis)
	}
	h.posts = make(map[string]*postEntry)
	h.store.Close()
}
