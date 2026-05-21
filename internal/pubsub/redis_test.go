package pubsub

import (
	"context"
	"testing"
	"time"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/config"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/redisclient"
)

func TestNew_StreamsStub(t *testing.T) {
	ps, err := New(config.PubSubStreams, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ps.Close()

	err = ps.Publish(context.Background(), "any", []byte("x"))
	if err != ErrNotImplemented {
		t.Fatalf("Publish err = %v, want ErrNotImplemented", err)
	}
}

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

	payload := []byte("hello-roundtrip")
	if err := ps.Publish(ctx, "test:pubsub_roundtrip", payload); err != nil {
		t.Fatal(err)
	}

	select {
	case m := <-ch:
		if m.Channel != "test:pubsub_roundtrip" {
			t.Fatalf("channel = %q", m.Channel)
		}
		if string(m.Payload) != string(payload) {
			t.Fatalf("payload = %q, want %q", m.Payload, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	if err := ps.Unsubscribe(ctx, "test:pubsub_roundtrip", ch); err != nil {
		t.Fatal(err)
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected consumer channel closed after Unsubscribe")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected closed channel to yield immediately")
	}
}
