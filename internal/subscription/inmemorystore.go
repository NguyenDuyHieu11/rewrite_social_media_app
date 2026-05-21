// Package subscription provides the in-memory fan-out layer for real-time
// delivery on a single gateway instance.
//
// # Role in the stack
//
// Each gateway process holds open SSE (and later WebSocket) connections in RAM.
// This store maps postID → connectionID → buffered Go channel. It does not
// know about Redis, HTTP, or other gateways — feed.Hub bridges Redis into
// this store via Publish, and HTTP handlers read from the per-connection
// channels returned by Subscribe.
//
// # Data model
//
//	subs[postID][connectionID] = { userID, ch }
//
// One user may have multiple connections to the same post (multiple tabs).
// Each gets a unique connectionID (UUID).
//
// # Drop policy
//
// Publish uses a non-blocking send (select/default). If a slow client's buffer
// is full, that client drops the event; others still receive it. This prevents
// one lagging browser from blocking the bridge goroutine.
//
// # Redis lifecycle
//
// Subscribe returns firstForPost / Unsubscribe returns lastForPost for callers
// that want to drive Redis directly. feed.Hub ignores those flags and uses its
// own refcount instead; the flags remain useful for tests and alternate designs.
package subscription

import (
	"sync"

	"github.com/google/uuid"
)

// Event is an opaque payload delivered to one subscriber. Serialization format
// is agreed between publisher (dispatcher) and consumer (SSE handler); the
// store does not parse it.
type Event = []byte

type subscription struct {
	userID string
	ch     chan Event
}

// Store holds all active real-time subscriptions on this gateway.
// Safe for concurrent use.
type Store struct {
	mu sync.RWMutex

	// subs maps postID → connectionID → subscription.
	subs map[string]map[string]*subscription
}

// New returns an empty Store ready for Subscribe calls.
func New() *Store {
	return &Store{
		subs: make(map[string]map[string]*subscription),
	}
}

// Subscribe registers ch as the delivery channel for one browser connection
// watching postID as userID.
//
// The caller owns ch; Store only sends to it. Buffer size is the caller's
// responsibility (Hub uses 64).
//
// Returns:
//   - connectionID — pass to Unsubscribe on disconnect
//   - firstForPost — true if this was the first subscription for postID on
//     this gateway (informational; Hub does not use it for Redis lifecycle)
func (s *Store) Subscribe(postID, userID string, ch chan Event) (string, bool) {
	connectionID := uuid.NewString()

	s.mu.Lock()
	defer s.mu.Unlock()

	inner, ok := s.subs[postID]

	var firstForPost bool
	if !ok {
		inner = make(map[string]*subscription)
		s.subs[postID] = inner
		firstForPost = true
	}

	inner[connectionID] = &subscription{
		userID: userID,
		ch:     ch,
	}

	return connectionID, firstForPost
}

// Unsubscribe removes the subscription identified by connectionID.
//
// userID is accepted for API symmetry and future validation; it is not
// currently checked against the stored subscription.
//
// Returns lastForPost — true if postID has no remaining subscriptions on this
// gateway after this call. Unknown connectionID is a no-op (returns false).
func (s *Store) Unsubscribe(postID, userID, connectionID string) (lastForPost bool) {
	_ = userID

	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.subs[postID]
	if !ok {
		return false
	}

	if _, exists := v[connectionID]; !exists {
		return false
	}

	delete(v, connectionID)

	if len(v) == 0 {
		delete(s.subs, postID)
		return true
	}
	return false
}

// Publish fans out ev to every channel registered under postID.
//
// Non-blocking: if a subscriber's channel buffer is full, the event is
// dropped for that subscriber only. Other subscribers still receive it.
// Called from feed.Hub.bridge when Redis delivers a message.
func (s *Store) Publish(postID string, ev Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inner, ok := s.subs[postID]
	if !ok {
		return
	}
	for _, sub := range inner {
		select {
		case sub.ch <- ev:
		default:
		}
	}
}

// Subscribers returns how many active connections are watching postID.
// Used by tests and metrics.
func (s *Store) Subscribers(postID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subs[postID])
}

// Close closes every subscriber channel and clears internal state.
// Invoke once during gateway shutdown after all HTTP handlers have drained.
// Do not call Subscribe after Close.
func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, inner := range s.subs {
		for _, sub := range inner {
			close(sub.ch)
		}
	}
	s.subs = make(map[string]map[string]*subscription)
}
