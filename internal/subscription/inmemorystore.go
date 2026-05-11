package subscription

import (
	"sync"

	"github.com/google/uuid"
)

type Event = []byte

type subscription struct {
	userID string
	ch     chan Event
}

type Store struct {
	mu sync.RWMutex

	// postID -> connectionID -> subscription
	subs map[string]map[string]*subscription
}

func New() *Store {
	return &Store{
		subs: make(map[string]map[string]*subscription),
	}
}

// Subscribe registers a new SSE channel for (postID, userID).
//
// Returns:
//
//	connectionID    — a unique opaque handle the caller passes to Unsubscribe.
//	firstForPost    — true iff this is now the only subscription for postID
//	                  on this gateway. The caller uses this to know it must
//	                  SUBSCRIBE to the Redis channel post:{postID}.
func (s *Store) Subscribe(postID, userID string, ch chan Event) (string, bool) {
	connectionID := uuid.NewString()

	s.mu.Lock()
	defer s.mu.Unlock()

	inner, ok := s.subs[postID]

	var firstForPost bool = false
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
// Returns:
//
//	lastForPost     — true iff postID now has zero subscriptions on this
//	                  gateway. The caller uses this to know it must
//	                  UNSUBSCRIBE from the Redis channel post:{postID}.
//
// If the connectionID does not exist, the call is a no-op (returns false).
func (s *Store) Unsubscribe(postID, userID, connectionID string) (lastForPost bool) {

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

// Publish fans out an event to every channel registered under postID.
//
// Non-blocking: if a subscriber's channel buffer is full, the event is
// dropped FOR THAT SUBSCRIBER ONLY. Other subscribers still receive it.
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
			// channel buffer full; drop for this subscriber only
		}
	}
}

// Subscribers returns the count of active subscriptions for postID.
// Used by tests and by gauges/metrics.
func (s *Store) Subscribers(postID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subs[postID])
}

// Close drains the store. Closes every subscriber channel and clears
// internal state. Called once during gateway shutdown.
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
