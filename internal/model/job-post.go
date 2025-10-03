package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// EditableJobPostInfo is part of job post that can be edited
type EditableJobPostInfo struct {
	Title    string         `gorm:"type:text" json:"title"`
	Desc     string         `gorm:"type:text" json:"desc"`
	Req      string         `gorm:"type:text" json:"req"`
	ExpLvl   string         `gorm:"type:text" json:"exp_lvl"`
	Location string         `gorm:"type:text" json:"location"`
	Type     string         `gorm:"type:text" json:"type"`
	Salary   string         `gorm:"type:text" json:"salary"`
	Tags     pq.StringArray `gorm:"type:text[]" json:"tags"`
	Expiring *time.Time     `gorm:"type:timestamp" json:"expiring,omitempty"`
}

// JobPost is gorm model for store job post data in DB
type JobPost struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;->" json:"id"`
	CompanyID uuid.UUID `gorm:"not null;index;<-:create" json:"company_id"`
	Company   Company   `gorm:"foreignKey:CompanyID;references:UserID" json:"-"`
	EditableJobPostInfo
	PostTime     time.Time     `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;->" json:"post_time"`
	Applications []Application `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE" json:"applications"`
}
