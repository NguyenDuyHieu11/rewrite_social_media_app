package subscription

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Single-thread sanity tests
// ---------------------------------------------------------------------------

func TestSubscribe_FirstAndSubsequent(t *testing.T) {
	s := New()
	ch1 := make(chan Event, 32)
	ch2 := make(chan Event, 32)

	conn1, first1 := s.Subscribe("post-1", "alice", ch1)
	if !first1 {
		t.Fatalf("first subscriber should be marked firstForPost; got false")
	}
	if conn1 == "" {
		t.Fatalf("connectionID must be non-empty")
	}

	conn2, first2 := s.Subscribe("post-1", "alice", ch2)
	if first2 {
		t.Fatalf("second subscriber should NOT be marked firstForPost; got true")
	}
	if conn2 == conn1 {
		t.Fatalf("two subscriptions must produce distinct connection IDs")
	}

	if got := s.Subscribers("post-1"); got != 2 {
		t.Fatalf("Subscribers = %d, want 2", got)
	}
}

func TestUnsubscribe_LastAndUnknown_NoOp(t *testing.T) {
	s := New()
	ch := make(chan Event, 32)

	// No-op on a post that doesn't exist.
	if got := s.Unsubscribe("ghost-post", "ghost-user", "ghost-conn"); got {
		t.Fatalf("Unsubscribe on unknown post should return false; got true")
	}

	conn, _ := s.Subscribe("post-1", "alice", ch)

	// No-op on a wrong connection ID.
	if got := s.Unsubscribe("post-1", "alice", "not-a-real-conn"); got {
		t.Fatalf("Unsubscribe with unknown connectionID should return false; got true")
	}
	if got := s.Subscribers("post-1"); got != 1 {
		t.Fatalf("Subscribers after bogus unsubscribe = %d, want 1", got)
	}

	// Real unsubscribe of the only entry → lastForPost.
	if got := s.Unsubscribe("post-1", "alice", conn); !got {
		t.Fatalf("Unsubscribe of last subscriber should return true; got false")
	}
	if got := s.Subscribers("post-1"); got != 0 {
		t.Fatalf("Subscribers after last unsubscribe = %d, want 0", got)
	}

	// Double-unsubscribe is a no-op.
	if got := s.Unsubscribe("post-1", "alice", conn); got {
		t.Fatalf("double Unsubscribe should return false; got true")
	}
}

func TestPublish_FanOutDeliversToAll(t *testing.T) {
	s := New()
	ch1 := make(chan Event, 4)
	ch2 := make(chan Event, 4)
	ch3 := make(chan Event, 4)

	s.Subscribe("post-1", "alice", ch1)
	s.Subscribe("post-1", "bob", ch2)
	s.Subscribe("post-1", "carol", ch3)

	s.Publish("post-1", []byte("hello"))

	for i, ch := range []chan Event{ch1, ch2, ch3} {
		select {
		case ev := <-ch:
			if string(ev) != "hello" {
				t.Fatalf("subscriber %d: got %q, want %q", i, ev, "hello")
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("subscriber %d did not receive event in time", i)
		}
	}
}

func TestPublish_NoSubscribersIsNoOp(t *testing.T) {
	s := New()

	// Should not panic, should not block.
	done := make(chan struct{})
	go func() {
		s.Publish("nobody-home", []byte("x"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Publish to a post with no subscribers blocked")
	}
}

func TestPublish_SlowConsumerDoesNotBlockOthers(t *testing.T) {
	s := New()

	// Slow consumer: buffer 1, never drained.
	chSlow := make(chan Event, 1)
	// Fast consumers: drained promptly.
	chFastA := make(chan Event, 8)
	chFastB := make(chan Event, 8)

	s.Subscribe("post-1", "slow", chSlow)
	s.Subscribe("post-1", "fastA", chFastA)
	s.Subscribe("post-1", "fastB", chFastB)

	const N = 5
	for i := 0; i < N; i++ {
		s.Publish("post-1", []byte("e"))
	}

	// Fast subscribers must have received all N events.
	for i, ch := range []chan Event{chFastA, chFastB} {
		for j := 0; j < N; j++ {
			select {
			case <-ch:
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("fast subscriber %d: missing event %d/%d", i, j+1, N)
			}
		}
	}

	// Slow subscriber's channel was buffer=1; it should contain exactly 1 event,
	// the rest dropped silently.
	if got := len(chSlow); got != 1 {
		t.Fatalf("slow subscriber should have buffered 1, got %d", got)
	}
}

func TestSubscribers_CountAndZero(t *testing.T) {
	s := New()
	if got := s.Subscribers("unknown"); got != 0 {
		t.Fatalf("Subscribers on unknown post = %d, want 0", got)
	}

	ch := make(chan Event, 1)
	s.Subscribe("post-1", "alice", ch)
	if got := s.Subscribers("post-1"); got != 1 {
		t.Fatalf("Subscribers after one subscribe = %d, want 1", got)
	}
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

func TestClose_DrainsChannelsAndResets(t *testing.T) {
	s := New()
	ch := make(chan Event, 4)
	s.Subscribe("post-1", "alice", ch)

	s.Close()

	// The channel must be closed: a receive yields ok=false promptly.
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("expected ch to be closed; receive returned ok=true")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected ch to be closed; receive blocked")
	}

	// After Close, the store is empty and re-usable.
	if got := s.Subscribers("post-1"); got != 0 {
		t.Fatalf("Subscribers after Close = %d, want 0", got)
	}
}

func TestClose_Idempotent(t *testing.T) {
	s := New()
	ch := make(chan Event, 1)
	s.Subscribe("post-1", "alice", ch)

	s.Close()
	// Calling Close again must not panic (channels are already drained;
	// the inner map was reset to empty).
	s.Close()

	// And the store must still be usable.
	ch2 := make(chan Event, 1)
	_, first := s.Subscribe("post-2", "bob", ch2)
	if !first {
		t.Fatalf("Subscribe after Close should mark first subscriber as firstForPost")
	}
}

// ---------------------------------------------------------------------------
// Concurrency tests — run with `go test -race`
// ---------------------------------------------------------------------------

// TestConcurrent_SubscribeUnsubscribeFinalStateZero spawns many goroutines that
// each subscribe and immediately unsubscribe. At the end, the post must have
// zero subscribers and the data structure must not have raced.
func TestConcurrent_SubscribeUnsubscribeFinalStateZero(t *testing.T) {
	s := New()
	defer s.Close()

	const goroutines = 100
	const itersPerG = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < itersPerG; i++ {
				ch := make(chan Event, 1)
				conn, _ := s.Subscribe("post-1", "u", ch)
				s.Unsubscribe("post-1", "u", conn)
			}
		}()
	}
	wg.Wait()

	if got := s.Subscribers("post-1"); got != 0 {
		t.Fatalf("final Subscribers = %d, want 0", got)
	}
}

// TestConcurrent_PublishWhileChurnNoDeadlock runs a publisher concurrently
// with many goroutines that churn (subscribe, drain a bit, unsubscribe).
// The test is bounded; the only failure modes are:
//   - deadlock (test exceeds the timer)
//   - race detected by `-race` (test fails automatically)
//   - panic (test fails)
func TestConcurrent_PublishWhileChurnNoDeadlock(t *testing.T) {
	s := New()
	defer s.Close()

	stop := make(chan struct{})
	var publishedCount atomic.Int64

	// Publisher goroutine: fires events as fast as it can until stop closes.
	var pubWG sync.WaitGroup
	pubWG.Add(1)
	go func() {
		defer pubWG.Done()
		for {
			select {
			case <-stop:
				return
			default:
				s.Publish("post-1", []byte("e"))
				publishedCount.Add(1)
			}
		}
	}()

	// Many churners: each subscribes, briefly drains, unsubscribes, repeats.
	const churners = 50
	const itersPerChurner = 20

	var churnWG sync.WaitGroup
	churnWG.Add(churners)
	for c := 0; c < churners; c++ {
		go func() {
			defer churnWG.Done()
			for i := 0; i < itersPerChurner; i++ {
				ch := make(chan Event, 4)
				conn, _ := s.Subscribe("post-1", "u", ch)

				// Drain whatever lands within a tiny window. Non-blocking
				// receive so we never hold the test up.
				drainBudget := 50
				for drainBudget > 0 {
					select {
					case <-ch:
						drainBudget--
					default:
						drainBudget = 0
					}
				}

				s.Unsubscribe("post-1", "u", conn)
			}
		}()
	}

	// Wait for churners to finish, then stop publisher.
	churnDone := make(chan struct{})
	go func() {
		churnWG.Wait()
		close(churnDone)
	}()

	select {
	case <-churnDone:
	case <-time.After(5 * time.Second):
		t.Fatalf("churners did not finish in 5s — possible deadlock")
	}

	close(stop)
	pubWG.Wait()

	// Final state must be clean.
	if got := s.Subscribers("post-1"); got != 0 {
		t.Fatalf("final Subscribers = %d, want 0", got)
	}
	if publishedCount.Load() == 0 {
		t.Fatalf("publisher never managed to publish anything; test is degenerate")
	}
	t.Logf("publisher fired %d events while churners ran", publishedCount.Load())
}

// TestConcurrent_PublishFanOutToManyDoesNotBlock guarantees that with many
// subscribers — even slow ones — Publish itself never blocks the caller for
// long. We measure wall-clock duration of N publishes against a generous
// upper bound. The point is the non-blocking property of select+default,
// not a precise latency bound.
func TestConcurrent_PublishFanOutToManyDoesNotBlock(t *testing.T) {
	s := New()
	defer s.Close()

	const subscribers = 100
	for i := 0; i < subscribers; i++ {
		// Tiny buffers; we never drain them. Publish must drop, not block.
		ch := make(chan Event, 1)
		s.Subscribe("post-1", "u", ch)
	}

	const publishes = 1000
	start := time.Now()
	for i := 0; i < publishes; i++ {
		s.Publish("post-1", []byte("e"))
	}
	elapsed := time.Since(start)

	// On a healthy machine this finishes in well under 100ms. The bound
	// here is loose to avoid flakiness on slow CI.
	if elapsed > 2*time.Second {
		t.Fatalf("Publish blocked: %d events to %d undrained subscribers took %v",
			publishes, subscribers, elapsed)
	}
	t.Logf("%d publishes to %d undrained subscribers in %v", publishes, subscribers, elapsed)
}
