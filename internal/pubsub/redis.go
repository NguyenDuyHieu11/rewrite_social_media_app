package pubsub

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

const defaultSubBuffer = 64

// RedisPubSub multiplexes all Redis subscriptions over one dedicated
// TCP connection and fans messages out to local consumer channels.
//
// Locking: mu serializes everything below it, INCLUDING the channel sends
// in fanOut. That is what makes Close safe to close consumer channels
// without racing the reader goroutine.
type RedisPubSub struct {
	client *redis.Client

	mu         sync.Mutex
	subs       map[string][]chan Message
	ps         *redis.PubSub // lazy: nil until first Subscribe
	closed     bool
	readerDone chan struct{} // lazy: nil until first Subscribe
}

func NewRedis(client *redis.Client) *RedisPubSub {
	return &RedisPubSub{
		client: client,
		subs:   make(map[string][]chan Message),
	}
}

func (r *RedisPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ErrClosed
	}
	r.mu.Unlock()

	if err := r.client.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("redis publish %q: %w", channel, err)
	}
	return nil
}

func (r *RedisPubSub) Subscribe(ctx context.Context, channel string) (<-chan Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, ErrClosed
	}

	if r.ps == nil {
		r.ps = r.client.Subscribe(ctx)
		r.readerDone = make(chan struct{})
		go r.reader()
	}

	if len(r.subs[channel]) == 0 {
		if err := r.ps.Subscribe(ctx, channel); err != nil {
			return nil, fmt.Errorf("redis subscribe %q: %w", channel, err)
		}
	}

	out := make(chan Message, defaultSubBuffer)
	r.subs[channel] = append(r.subs[channel], out)
	return out, nil
}

func (r *RedisPubSub) Unsubscribe(ctx context.Context, channel string, sub <-chan Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrClosed
	}

	list := r.subs[channel]
	found := -1
	for i, ch := range list {
		// chan Message and <-chan Message compare equal after this cast
		// when they reference the same underlying channel.
		if (<-chan Message)(ch) == sub {
			found = i
			break
		}
	}
	if found < 0 {
		return nil
	}

	close(list[found])
	list = append(list[:found], list[found+1:]...)

	if len(list) == 0 {
		delete(r.subs, channel)
		if err := r.ps.Unsubscribe(ctx, channel); err != nil {
			return fmt.Errorf("redis unsubscribe %q: %w", channel, err)
		}
		return nil
	}
	r.subs[channel] = list
	return nil
}

func (r *RedisPubSub) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true

	for _, list := range r.subs {
		for _, target := range list {
			close(target)
		}
	}
	r.subs = nil

	ps := r.ps
	readerDone := r.readerDone
	r.mu.Unlock()

	if ps != nil {
		_ = ps.Close()
		<-readerDone
	}
	return nil
}

func (r *RedisPubSub) reader() {
	defer close(r.readerDone)         //only works for bidirectional or send only channel
	for msg := range r.ps.Channel() { // loop through incoming message from redis
		r.fanOut(msg) //fan out the message
	}
}

func (r *RedisPubSub) fanOut(msg *redis.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}
	out := Message{Channel: msg.Channel, Payload: []byte(msg.Payload)}
	for _, t := range r.subs[msg.Channel] {
		select {
		case t <- out:
		default:
		}
	}
}
