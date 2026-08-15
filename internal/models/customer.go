package models

import (
	"time"

	"gorm.io/gorm"
)

// Customer represents the central user account entity.
type Customer struct {
	ID                uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Username          string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	Password          string         `gorm:"type:varchar(255);not null" json:"-"`
	Email             string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Phone             string         `gorm:"type:varchar(20)" json:"phone"`
	Name              string         `gorm:"type:varchar(100)" json:"name"`
	Surname           string         `gorm:"type:varchar(100)" json:"surname"`
	Birthday          *time.Time     `json:"birthday"`
	Gender            string         `gorm:"type:varchar(20)" json:"gender"`
	ProfilePictureUrl string         `gorm:"type:varchar(500)" json:"profile_picture_url"`
	Biography         string         `gorm:"type:text" json:"biography"`
	ComplaintCount    int            `gorm:"default:0" json:"complaint_count"`
	ContentCount      int            `gorm:"default:0" json:"content_count"`
	FollowingCount    int            `gorm:"default:0" json:"following_count"`
	FollowersCount    int            `gorm:"default:0" json:"followers_count"`
	PrivateAccount    bool           `gorm:"default:false" json:"private_account"`
	AccountStatus     string         `gorm:"type:varchar(50);default:'active'" json:"account_status"`
	RegistrationDate  time.Time      `gorm:"autoCreateTime" json:"registration_date"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Contents  []Content `gorm:"foreignKey:UserID" json:"contents,omitempty"`
	Comments  []Comment `gorm:"foreignKey:UserID" json:"comments,omitempty"`
	Likes     []Liked   `gorm:"foreignKey:UserID" json:"likes,omitempty"`
	Following []Follow  `gorm:"foreignKey:FollowerID" json:"following,omitempty"`
	Followers []Follow  `gorm:"foreignKey:FollowedUserID" json:"followers,omitempty"`
}
