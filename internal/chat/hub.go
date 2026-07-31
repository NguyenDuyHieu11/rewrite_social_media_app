package chat

import (
	"context"
	"sync"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/channels"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/pubsub"
)

type Hub struct {
	mu sync.Mutex

	bus   pubsub.PubSub
	store *WSStore
	rooms map[string]*roomEntry
}

type roomEntry struct {
	refs   int
	redis  <-chan pubsub.Message
	cancel context.CancelFunc
}

func NewHub(bus pubsub.PubSub, store *WSStore) *Hub {
	return &Hub{
		bus:   bus,
		store: store,
		rooms: make(map[string]*roomEntry),
	}
}

func (h *Hub) JoinRoom(ctx context.Context, roomID, userID string) error {
	_ = ctx
	h.mu.Lock()
	defer h.mu.Unlock()
	added := h.store.JoinRoom(roomID, userID)
	if !added {
		return nil
	}

	entry := h.rooms[roomID]
	if entry == nil {
		bridgeCtx, cancel := context.WithCancel(context.Background())
		redisCh, err := h.bus.Subscribe(bridgeCtx, channels.RoomChannel(roomID))
		if err != nil {
			cancel()
			h.store.LeaveRoom(roomID, userID)
			return err
		}
		entry = &roomEntry{refs: 0, redis: redisCh, cancel: cancel}
		h.rooms[roomID] = entry
		go h.bridge(bridgeCtx, roomID, redisCh)
	}
	entry.refs++
	return nil
}

func (h *Hub) LeaveRoom(ctx context.Context, roomID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	removed, roomEmpty := h.store.LeaveRoom(roomID, userID)
	if !removed {
		return
	}
	entry := h.rooms[roomID]
	if entry == nil {
		return
	}
	entry.refs--
	if !roomEmpty {
		return
	}
	entry.cancel()
	_ = h.bus.Unsubscribe(ctx, channels.RoomChannel(roomID), entry.redis)
	delete(h.rooms, roomID)
}

func (h *Hub) LeaveAllForUser(ctx context.Context, userID string) {
	rooms := h.store.RoomsForUser(userID)
	for _, roomID := range rooms {
		h.LeaveRoom(ctx, roomID, userID)
	}
}
func (h *Hub) bridge(ctx context.Context, roomID string, redisCh <-chan pubsub.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-redisCh:
			if !ok {
				return
			}
			h.store.DeliverToRoom(roomID, msg.Payload)
		}
	}
}
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for roomID, entry := range h.rooms {
		entry.cancel()
		_ = h.bus.Unsubscribe(context.Background(), channels.RoomChannel(roomID), entry.redis)
	}
	h.rooms = make(map[string]*roomEntry)
	h.store.Close()
}
