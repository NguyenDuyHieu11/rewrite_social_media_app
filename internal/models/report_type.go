package domain

// ReportType defines predefined categories and SLAs for moderation reports.
type ReportType struct {
	ID               uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryName     string `gorm:"type:varchar(100);not null" json:"category_name"`
	Name             string `gorm:"type:varchar(100);not null" json:"name"`
	Importance       int    `gorm:"default:1" json:"importance"`
	DeadlineDuration int    `gorm:"comment:deadline duration in hours" json:"deadline_duration"`

	// Relationships
	Reports []Report `gorm:"foreignKey:ReportTypeID" json:"reports,omitempty"`
}
