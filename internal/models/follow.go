package models

import (
	"time"

	"github.com/google/uuid"
)

// Follow is a directed edge follower_id → followee_id in the social graph.
// The home feed is follow-only: timeline rows come from user_feed populated
// from this graph (fan-out on write, implemented in a later phase).
type Follow struct {
	FollowerID uuid.UUID `json:"follower_id"`
	FolloweeID uuid.UUID `json:"followee_id"`
	CreatedAt  time.Time `json:"created_at"`
}
