package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID
}

// IsActive returns true if the token has not expired and has not been revoked.
func (t *RefreshToken) IsActive(now time.Time) bool {
	if t.RevokedAt != nil {
		return false
	}
	return now.Before(t.ExpiresAt)
}
