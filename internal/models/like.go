package models

import (
	"time"
)

// Liked tracks user likes and engagement metrics across contents, comments, and hashtags.
type Liked struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	Customer        Customer  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	LikedContentID  *uint     `gorm:"index" json:"liked_content_id"`
	Content         *Content  `gorm:"foreignKey:LikedContentID;constraint:OnDelete:SET NULL" json:"content,omitempty"`
	LikedCommentID  *uint     `gorm:"index" json:"liked_comment_id"`
	Comment         *Comment  `gorm:"foreignKey:LikedCommentID;constraint:OnDelete:SET NULL" json:"comment,omitempty"`
	HashtagID       *uint     `gorm:"index" json:"hashtag_id"`
	TimeSpent       int       `gorm:"default:0;comment:time spent in seconds" json:"time_spent"`
	ActionFrequency int       `gorm:"default:1" json:"action_frequency"`
	CreationDate    time.Time `gorm:"autoCreateTime" json:"creation_date"`
}
