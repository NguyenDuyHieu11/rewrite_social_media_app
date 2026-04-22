# Project Prompt — Social Media Web App

## Overview

Build a social media web application in Go. The backend is the primary concern. The frontend stack is your choice — pick whatever integrates cleanly with SSE and WebSockets. The project is structured as a monorepo with two Go binaries under `cmd/`.

---

## Architecture

This project follows the real-time platform design described by Neo Kim on systemdesign.one, specifically the live comment system design and the real-time presence platform design. The architecture must faithfully implement the patterns described in those articles, adapted for a single-datacenter deployment without distributed infrastructure.

### The two binaries

```
cmd/
  gateway/      — holds SSE and WebSocket connections to clients, delivers events
  dispatcher/   — receives published events, routes them to the correct gateway servers
```

They communicate over HTTP. The dispatcher queries the endpoint store to find which gateway servers have subscribers for a given resource, then forwards the event to those gateways over HTTP. Each gateway then checks its own in-memory subscription store and delivers to the relevant connections.

This separation must be real and enforced — the gateway and dispatcher must not call each other's internal packages directly. All cross-binary communication goes over HTTP.

### Shared packages

Common types, interfaces, and utilities live in `internal/` and are imported by both binaries.

```
internal/
  pubsub/         — pub/sub interface and implementations
  subscription/   — subscription store logic
  presence/       — presence service logic
  store/          — endpoint store logic
  auth/           — JWT + refresh token logic
  rbac/           — role-based access control
  models/         — shared domain types (User, Post, Comment, Message, etc.)
  middleware/      — HTTP middleware (auth, RBAC, logging)
```

---

## Application domains

The application has two completely distinct domains accessible from two separate areas of the UI — a home feed and a chat page. They use different real-time transports and have no awareness of each other's state.

```
Taskbar
├── Home (feed icon)   — post activity domain — SSE transport
└── Chat (chat icon)   — messaging domain     — WebSocket transport
```

A user on the home feed has an SSE connection open. A user who opens the chat page additionally opens a WebSocket connection. Both can be active simultaneously. They are independent.

---

## Features in Scope

### Domain 1 — Home feed (SSE)

#### 1. Regular posting
- Users can create posts containing text, images, and uploaded videos
- Images and videos are stored in object storage — use MinIO locally (S3-compatible, free, self-hosted) to simulate Cloudflare R2 or similar
- Posts are persisted in PostgreSQL
- Post creation, editing, deletion are standard CRUD via HTTP REST endpoints
- Likes, comments, and reactions on posts are CRUD on the write side, real-time on the delivery side

#### 2. Real-time post activity
- When a user opens a post, their client opens an SSE connection to the gateway and subscribes to that post's channel
- New comments, reactions, and likes on that post are delivered in real time over SSE to all subscribed clients without page refresh
- When the user navigates away, the SSE connection closes and the subscription is torn down
- The SSE connection carries only post-domain event types distinguished by the `event:` field:
  - `comment` — new comment on a subscribed post
  - `reaction` — new reaction on a subscribed post
- Presence is not part of the home feed domain — no online/offline indicators appear here

---

### Domain 2 — Chat page (WebSocket)

The chat page functions like Facebook Messenger. It is entirely WebSocket-driven. When a user navigates to the chat page, a single WebSocket connection is opened and maintained for the entire session on that page. All chat functionality flows through this connection.

#### 3. Chat (direct messaging and group messaging)
- Users can send direct messages to one other user or to a group of users
- Each conversation (direct or group) is a chat room identified by a room ID
- Sending a message goes over the WebSocket connection
- Messages are persisted in PostgreSQL
- The dispatcher routes incoming messages to the correct gateway server holding the recipient's WebSocket connection
- Message history is loaded via a standard HTTP REST endpoint on page open, then new messages arrive via WebSocket from that point forward

#### 4. Presence (online/offline status) — chat domain only
- Presence is a feature of the chat page exclusively — online/offline indicators appear only in the chat contact list and inside chat rooms, not on the home feed
- When a client opens the chat page and establishes a WebSocket connection, the gateway registers a heartbeat for that user in the presence service
- The presence service records the user as online in Redis: `HSET presence:{userID} last_seen {unix_timestamp}` with a TTL slightly longer than the heartbeat interval
- A goroutine (one per online user) holds a timer that resets on each heartbeat — if the timer fires, the user is considered offline
- On offline detection, publish an offline event through the pub/sub layer so all users who share a chat room with the offline user receive a `presence_offline` WebSocket event
- On online detection, publish a `presence_online` event to the same audience
- Display last active timestamp for offline users inside the chat UI
- When a client closes the chat page or disconnects, detect it via WebSocket close event and trigger offline detection immediately rather than waiting for the timer
- Heartbeat is sent by the client over the WebSocket connection — no separate HTTP heartbeat endpoint needed

#### WebSocket event types (chat domain)
All WebSocket messages are JSON with a `type` field for routing on both client and server:
- `message` — a new chat message in a room
- `presence_online` — a contact came online
- `presence_offline` — a contact went offline
- `typing` — a contact is typing in the current room
- `seen` — a contact has seen a message

---

## Data Storage

### PostgreSQL
Persistent storage for all durable data:
- Users
- Posts (text, image URLs, video URLs)
- Comments
- Reactions and likes
- Direct messages
- Follow relationships
- Roles and permissions (for RBAC)

### Redis
Used for three distinct purposes — use separate key namespaces for each:

**1. Presence state**
```
HSET presence:{userID} last_seen {unix_timestamp}
```
With TTL set to heartbeat interval + buffer.

**2. Pub/sub layer**
The pub/sub layer must be implemented behind an interface so the concrete implementation can be swapped:

```go
type PubSub interface {
    Publish(ctx context.Context, channel string, payload []byte) error
    Subscribe(ctx context.Context, channel string) (<-chan []byte, error)
    Unsubscribe(ctx context.Context, channel string) error
}
```

Provide two concrete implementations:
- `RedisPubSub` — uses Redis pub/sub (`PUBLISH`, `SUBSCRIBE`)
- `RedisStreamsPubSub` — uses Redis Streams (`XADD`, `XREAD` with blocking)

Wire up `RedisPubSub` as the default. The choice between them is deferred — swapping is a one-line change at the composition root.

**3. Recent comments cache**
```
ZADD post:{postID}:comments {timestamp} {commentJSON}
```
Store the latest 50 comments per post as a sorted set. Serve from cache on post open, fall back to PostgreSQL for older comments.

### Object storage — MinIO
Run MinIO locally as a self-hosted S3-compatible store for images and videos. The Go backend uploads files to MinIO and stores the resulting URL in PostgreSQL. Do not serve media files through the Go server.

---

## Subscription Architecture

### In-memory subscription store (inside gateway)
Each gateway instance holds two separate in-memory maps — one for SSE (home feed domain) and one for WebSocket (chat domain). They are completely independent and must not be mixed.

**SSE subscription store — home feed domain**
```go
// postID -> userID -> SSE channel
map[string]map[string]chan Event
```

**WebSocket connection store — chat domain**
```go
// userID -> WebSocket connection
map[string]*websocket.Conn

// roomID -> set of userIDs in that room
map[string]map[string]struct{}
```

Both are protected by `sync.RWMutex`. The SSE store answers "which SSE connections on this gateway are subscribed to post:99?" The WebSocket store answers "which WebSocket connections on this gateway belong to users in room:42?"

### Endpoint store (shared, persisted in Redis)
Maps resource channels to the set of gateway servers that have active subscribers for them:

```go
// "post:99" -> {"gateway-1", "gateway-3"}
SADD endpoint:{channel} {gatewayID}
SREM endpoint:{channel} {gatewayID}
SMEMBERS endpoint:{channel}
```

This is the coarse-grained routing layer — it answers "which gateway servers need to receive events for post:99?" The dispatcher queries this before forwarding events. Gateway servers register and deregister themselves here as subscriptions are created and torn down.

### Subscribe lifecycle — SSE (home feed)
1. Client opens SSE connection with `Accept: text/event-stream` via HTTP PUT to `/posts/:postID/subscriptions`
2. Gateway adds the connection to its in-memory SSE subscription store
3. Gateway registers itself in the endpoint store: `SADD endpoint:post:{postID} {gatewayID}`
4. Client navigates away — SSE connection closes
5. Gateway detects closure via `r.Context().Done()`
6. Gateway removes connection from in-memory SSE store
7. If no more connections remain for that post on this gateway, gateway deregisters from endpoint store: `SREM endpoint:post:{postID} {gatewayID}`

### Subscribe lifecycle — WebSocket (chat)
1. Client opens WebSocket connection via `GET /ws/chat` with `Upgrade: websocket` header
2. Gateway upgrades the connection and registers the user's WebSocket connection in its in-memory WebSocket store
3. Gateway registers itself in the endpoint store for each room the user belongs to: `SADD endpoint:room:{roomID} {gatewayID}`
4. Client sends a heartbeat over the WebSocket connection every 30 seconds — gateway resets the presence timer on receipt
5. Client closes the chat page — WebSocket close event fires
6. Gateway removes the connection from its in-memory WebSocket store
7. Gateway deregisters from the endpoint store for all rooms the user was in
8. Gateway triggers immediate offline detection for the user

---

## Publish flow — home feed (comment, reaction, like)

```
Client submits comment via HTTP POST
→ dispatcher receives it
→ writes comment to PostgreSQL
→ writes to Redis pub/sub (or Streams) on channel post:{postID}
→ queries endpoint store: SMEMBERS endpoint:post:{postID}
→ forwards event to each listed gateway over HTTP
→ each gateway queries its in-memory SSE subscription store
→ each gateway writes event to all subscribed SSE channels
→ clients receive event, DOM updates without refresh
```

## Publish flow — chat (message, presence, typing, seen)

```
Client sends message over WebSocket
→ gateway receives it
→ writes message to PostgreSQL
→ writes to Redis pub/sub (or Streams) on channel room:{roomID}
→ queries endpoint store: SMEMBERS endpoint:room:{roomID}
→ forwards event to each listed gateway over HTTP
→ each gateway queries its in-memory WebSocket store for users in that room
→ each gateway writes event to all relevant WebSocket connections
→ recipients receive message in real time
```

Presence events follow the same chat publish flow but are triggered by the presence service rather than a client action — on timer fire (offline) or on WebSocket connect (online).

---

## Authentication

- JWT access tokens — short-lived (15 minutes)
- Refresh tokens — long-lived (7 days), stored in PostgreSQL, rotated on use
- All endpoints require a valid JWT except registration and login
- SSE and WebSocket connections authenticate via JWT passed as a query parameter or Authorization header on the initial HTTP request

---

## RBAC

Role-based access control. Define roles in PostgreSQL. Middleware checks the user's role before allowing access to protected routes. At minimum define:
- `admin` — full access
- `user` — standard access
- `moderator` — can delete posts and comments from others

RBAC checks happen in HTTP middleware before the handler is reached. The middleware reads the user's role from the JWT claims.

---

## Heartbeat and presence timing

Presence is a chat domain concern only. Heartbeats are sent by the client over the WebSocket connection — no separate HTTP heartbeat endpoint.

- Client sends a heartbeat JSON message `{"type": "heartbeat"}` over WebSocket every 30 seconds
- Presence TTL in Redis: 45 seconds (heartbeat interval + 50% buffer)
- Offline detection timer: fires after 45 seconds of no heartbeat
- On heartbeat arrival: reset the goroutine timer, update Redis TTL
- On timer fire: publish `presence_offline` event to all rooms the user belongs to, clean up presence state
- On WebSocket close: trigger offline detection immediately, do not wait for timer

---

## SSE event format — home feed domain

All SSE events follow this structure over the wire:

```
id: {eventID}
event: {eventType}
data: {jsonPayload}

```

The blank line terminates the event. SSE event types: `comment`, `reaction`.

The `retry:` field should be set to `3000` (3 seconds) on the initial connection response so clients automatically reconnect after a dropped connection.

## WebSocket message format — chat domain

All WebSocket messages are JSON with a `type` field for client-side routing:

```json
{
  "type": "message",
  "payload": {
    "room_id": "42",
    "user_id": "99",
    "text": "hello",
    "created_at": "2026-04-21T10:00:00Z"
  }
}
```

WebSocket message types: `message`, `presence_online`, `presence_offline`, `typing`, `seen`.

---

## Non-functional requirements to implement correctly

- SSE connections are non-blocking — writing to a slow client must not block delivery to other clients. Use buffered channels.
- WebSocket connections follow the same non-blocking principle.
- All concurrent access to the in-memory subscription stores must go through `sync.RWMutex` — reads use `RLock`, writes use `Lock`.
- The gateway detects dead SSE connections via `r.Context().Done()` and cleans up immediately.
- The gateway detects dead WebSocket connections via the WebSocket close event and cleans up immediately including triggering offline detection.
- The dispatcher is stateless — it holds no connections and no subscription state. It only queries the endpoint store and forwards.
- The gateway exposes two distinct route groups — `/sse/*` for the home feed domain and `/ws/*` for the chat domain. Protocol detection is explicit via separate routes, not implicit header inspection.
- All database queries use parameterized statements — no string interpolation in SQL.
- Structured logging throughout using `log/slog` (Go standard library).

---

## Project structure

```
/
├── go.mod
├── go.sum
├── cmd/
│   ├── gateway/
│   │   └── main.go
│   └── dispatcher/
│       └── main.go
├── internal/
│   ├── models/
│   ├── pubsub/
│   │   ├── interface.go
│   │   ├── redis_pubsub.go
│   │   └── redis_streams.go
│   ├── subscription/
│   │   ├── sse_store.go       — SSE subscription store (home feed domain)
│   │   └── ws_store.go        — WebSocket connection store (chat domain)
│   ├── presence/
│   ├── store/
│   │   └── endpoint_store.go
│   ├── auth/
│   ├── rbac/
│   └── middleware/
├── migrations/
│   └── *.sql
└── docker-compose.yml
```

---

## Docker Compose

Provide a `docker-compose.yml` that runs:
- PostgreSQL
- Redis (single instance, AOF enabled)
- MinIO

The Go binaries run outside Docker during development and connect to these services locally.

---

## What not to build

- No multi-datacenter deployment
- No Kafka or Apache Cassandra
- No GeoDNS or CDN configuration
- No live video streaming
- No HyperLogLog for comment counts — use a simple Redis counter or PostgreSQL COUNT
- No serverless functions
- No actor frameworks — Go goroutines handle concurrency natively
