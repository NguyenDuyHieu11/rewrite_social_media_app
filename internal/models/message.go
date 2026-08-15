package domain

import (
	"time"

	"gorm.io/gorm"
)

// Conversation represents a direct message thread or group conversation.
type Conversation struct {
	ID                  uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	FirstUserID         uint           `gorm:"not null;index" json:"first_user_id"`
	FirstUser           Customer       `gorm:"foreignKey:FirstUserID;constraint:OnDelete:CASCADE" json:"-"`
	SecondUserID        uint           `gorm:"not null;index" json:"second_user_id"`
	SecondUser          Customer       `gorm:"foreignKey:SecondUserID;constraint:OnDelete:CASCADE" json:"-"`
	LastMessageID       *uint          `gorm:"index" json:"last_message_id"`
	LastMessage         *Message       `gorm:"foreignKey:LastMessageID;constraint:OnDelete:SET NULL" json:"last_message,omitempty"`
	LastMessageContent  string         `gorm:"type:text" json:"last_message_content"`
	LastMessageSentTime *time.Time     `json:"last_message_sent_time"`
	CreationDate        time.Time      `gorm:"autoCreateTime" json:"creation_date"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Messages []Message `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}
