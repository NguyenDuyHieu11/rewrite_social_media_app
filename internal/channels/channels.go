// Package channels defines stable Redis pub/sub channel names used across
// binaries (gateway and dispatcher).
//
// Centralizing names here prevents typos like "post:42" vs "posts:42" and
// keeps the dispatcher's PUBLISH targets aligned with the gateway's SUBSCRIBE
// targets. Only naming lives in this package — no I/O, no business logic.
//
// Naming convention:
//
//	post:{uuid}   — home-feed events for one post (comments, reactions, etc.)
//
// Chat domain channels (room:{id}) will be added in a later milestone when
// WebSocket messaging is implemented.
package channels

import "fmt"

// PostChannel returns the Redis channel name for real-time home-feed events
// on the given post. postID is the Postgres UUID string (or any opaque id
// both sides agree on).
//
// Example: PostChannel("550e8400-e29b-41d4-a716-446655440000") → "post:550e8400-..."
//
// Publishers (dispatcher, after M10) and subscribers (gateway feed.Hub) must
// both use this helper so messages route correctly.
func PostChannel(postID string) string {
	return fmt.Sprintf("post:%s", postID)
}
