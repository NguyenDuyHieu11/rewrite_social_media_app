package domain

import (
	"time"

	"gorm.io/gorm"
)

// Content represents media, posts, or articles published by a Customer.
type Content struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint           `gorm:"not null;index" json:"user_id"`
	Customer     Customer       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Type         string         `gorm:"type:varchar(50);not null" json:"type"`
	ContentURL   string         `gorm:"type:varchar(500)" json:"content_url"`
	SpanInfo     string         `gorm:"type:varchar(255)" json:"span_info"`
	TargetGender string         `gorm:"type:varchar(20)" json:"target_gender"`
	ViewCount    int64          `gorm:"default:0" json:"view_count"`
	ShowCount    int64          `gorm:"default:0" json:"show_count"`
	CommentCount int64          `gorm:"default:0" json:"comment_count"`
	RepostCount  int64          `gorm:"default:0" json:"repost_count"`
	CreationDate time.Time      `gorm:"autoCreateTime" json:"creation_date"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Comments []Comment `gorm:"foreignKey:ContentID" json:"comments,omitempty"`
	Likes    []Liked   `gorm:"foreignKey:LikedContentID" json:"likes,omitempty"`
}
