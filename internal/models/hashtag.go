package domain

// Hashtag represents tags attached to content posts or chat messages.
type Hashtag struct {
	ID                uint             `gorm:"primaryKey;autoIncrement" json:"id"`
	HashtagCategoryID *uint            `gorm:"index" json:"hashtag_category_id"`
	HashtagCategory   *HashtagCategory `gorm:"foreignKey:HashtagCategoryID;constraint:OnDelete:SET NULL" json:"hashtag_category,omitempty"`
	Name              string           `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	MatchingCategory  string           `gorm:"type:varchar(100)" json:"matching_category"`
	PopularityScore   int              `gorm:"default:0" json:"popularity_score"`
}
