package models

import (
	"time"
)

// Follow represents the self-referential many-to-many relationship between Customers.
type Follow struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FollowerID     uint      `gorm:"not null;index:idx_follow_pair,unique" json:"follower_id"`
	Follower       Customer  `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE" json:"-"`
	FollowedUserID uint      `gorm:"not null;index:idx_follow_pair,unique" json:"followed_user_id"`
	FollowedUser   Customer  `gorm:"foreignKey:FollowedUserID;constraint:OnDelete:CASCADE" json:"-"`
	FollowStatus   string    `gorm:"type:varchar(50);default:'pending'" json:"follow_status"`
	FollowStart    time.Time `gorm:"autoCreateTime" json:"follow_start"`
	FollowDate     time.Time `gorm:"autoCreateTime" json:"follow_date"`
}
