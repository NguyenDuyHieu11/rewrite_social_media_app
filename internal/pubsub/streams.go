package pubsub

import "context"

// StreamsPubSub is a placeholder for a future Redis Streams backend.
type StreamsPubSub struct{}

func NewStreams() *StreamsPubSub {
	return &StreamsPubSub{}
}

func (*StreamsPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	return ErrNotImplemented
}

func (*StreamsPubSub) Subscribe(ctx context.Context, channel string) (<-chan Message, error) {
	return nil, ErrNotImplemented
}

func (*StreamsPubSub) Unsubscribe(ctx context.Context, channel string, sub <-chan Message) error {
	return ErrNotImplemented
}

func (*StreamsPubSub) Close() error {
	return nil
}
