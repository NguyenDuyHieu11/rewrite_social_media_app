package pubsub

import (
	"context"
	"testing"
	"time"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/redisclient"
)

func TestRedisPubSub_PublishSubscribe(t *testing.T) {
	ctx := context.Background()
	client, err := redisclient.New(ctx, "localhost:6379", "", 0)
	if err != nil {
		t.Skip("redis not available:", err)
	}
	defer client.Close()

	ps := NewRedis(client)
	defer ps.Close()

	ch, err := ps.Subscribe(ctx, "test:pubsub_roundtrip")
	if err != nil {
		t.Fatal(err)
	}

	// Redis pub/sub is fire-and-forget: a PUBLISH that lands before the
	// SUBSCRIBE takes effect server-side is dropped, and Subscribe does not
	// wait for server confirmation. Republish until the subscription is live
	// instead of racing a single publish against it.
	payload := []byte("hello-roundtrip")
	deadline := time.After(2 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()

	if err := ps.Publish(ctx, "test:pubsub_roundtrip", payload); err != nil {
		t.Fatal(err)
	}
recv:
	for {
		select {
		case m := <-ch:
			if m.Channel != "test:pubsub_roundtrip" {
				t.Fatalf("channel = %q", m.Channel)
			}
			if string(m.Payload) != string(payload) {
				t.Fatalf("payload = %q, want %q", m.Payload, payload)
			}
			break recv
		case <-tick.C:
			if err := ps.Publish(ctx, "test:pubsub_roundtrip", payload); err != nil {
				t.Fatal(err)
			}
		case <-deadline:
			t.Fatal("timeout waiting for message")
		}
	}

	if err := ps.Unsubscribe(ctx, "test:pubsub_roundtrip", ch); err != nil {
		t.Fatal(err)
	}

	// Drain any duplicate messages from the retry loop, then expect close.
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // closed as expected
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected closed channel to yield immediately")
		}
	}
}
