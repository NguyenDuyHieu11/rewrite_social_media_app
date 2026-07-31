package chat

import "sync"

type Event = []byte
type client struct {
	userID string
	send   chan Event
}
type WSStore struct {
	mu        sync.RWMutex
	clients   map[string]*client
	rooms     map[string]map[string]struct{} // store all room in key, each room has its user in value.
	userRooms map[string]map[string]struct{} // store all user in key, each value is the room the user is in
}

func NewWSStore() *WSStore {
	return &WSStore{
		clients:   make(map[string]*client),
		rooms:     make(map[string]map[string]struct{}),
		userRooms: make(map[string]map[string]struct{}),
	}
}

// policy: remove old connection by closing old.send if he already registered
func (s *WSStore) Register(userID string) chan Event {
	ch := make(chan Event, 64)

	s.mu.Lock()
	defer s.mu.Unlock()

	if old, ok := s.clients[userID]; ok {
		close(old.send)
	}

	s.clients[userID] = &client{userID: userID, send: ch}
	return ch
}

func (s *WSStore) Unregister(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[userID]
	if !ok {
		return
	}
	delete(s.clients, userID)
	close(c.send)
}

func (s *WSStore) JoinRoom(roomID, userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[userID]; !ok {
		return false
	}
	if ur := s.userRooms[userID]; ur != nil {
		if _, already := ur[roomID]; already {
			return false
		}
	} else {
		s.userRooms[userID] = make(map[string]struct{})
	}
	members, ok := s.rooms[roomID] // xem xem key nay cos ton tai khong
	if !ok {
		members = make(map[string]struct{})
		s.rooms[roomID] = members
	}
	members[userID] = struct{}{}
	s.userRooms[userID][roomID] = struct{}{}
	return true
}

func (s *WSStore) LeaveRoom(roomID, userID string) (removed bool, roomEmpty bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.rooms[roomID]
	if !ok {
		return false, false
	}
	if _, ok := members[userID]; !ok {
		return false, false
	}
	delete(members, userID)
	removed = true
	if len(members) == 0 {
		delete(s.rooms, roomID)
		roomEmpty = true
	}
	if ur, ok := s.userRooms[userID]; ok {
		delete(ur, roomID)
		if len(ur) == 0 {
			delete(s.userRooms, userID)
		}
	}
	return removed, roomEmpty
}

func (s *WSStore) RoomsForUser(userID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ur := s.userRooms[userID]
	out := make([]string, 0, len(ur))
	for roomID := range ur {
		out = append(out, roomID)
	}
	return out
}

func (s *WSStore) DeliverToRoom(roomID string, payload Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for userID := range s.rooms[roomID] {
		c := s.clients[userID] // tat ca nguoi nhan cua minh trong roomID
		if c == nil {
			continue
		}

		select {
		case c.send <- payload:
		default:
		}
	}
}

func (s *WSStore) Members(roomID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rooms[roomID])
}

func (s *WSStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for userID, c := range s.clients {
		close(c.send)
		delete(s.clients, userID)
	}

	s.rooms = make(map[string]map[string]struct{})
	s.userRooms = make(map[string]map[string]struct{})
}
