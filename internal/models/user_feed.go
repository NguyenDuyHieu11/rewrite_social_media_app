package models

import (
	"time"

	"github.com/google/uuid"
)

// UserFeedEntry is one row in the materialized per-user timeline (fan-out table).
//
// Read path (follow-only feed, no ranking algorithm):
//   SELECT p.* FROM user_feed uf
//   JOIN posts p ON p.id = uf.post_id
//   WHERE uf.user_id = $viewer
//     AND ($cursor_rank IS NULL OR (uf.rank_at, uf.post_id) < ($cursor_rank, $cursor_post))
//   ORDER BY uf.rank_at DESC, uf.post_id DESC
//   LIMIT $limit
//
// Write path (future worker): when author publishes, INSERT one row per follower
// (+ optionally the author). Avoid OFFSET pagination on this query — use the
// (rank_at, post_id) cursor tuple to prevent drift while new posts arrive.
type UserFeedEntry struct {
	UserID    uuid.UUID `json:"user_id"`
	PostID    uuid.UUID `json:"post_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	RankAt    time.Time `json:"rank_at"`
	CreatedAt time.Time `json:"created_at"`
}

// FeedCursor is the keyset pagination token for user_feed queries.
// Encode as opaque string in API responses (Phase B).
type FeedCursor struct {
	RankAt time.Time `json:"rank_at"`
	PostID uuid.UUID `json:"post_id"`
}
