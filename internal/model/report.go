package model

import "github.com/google/uuid"

var (
	ReportStatusPending   = "pending"
	ReportStatusResolved  = "resolved"
	ReportStatusRejected  = "rejected"
)

type ReportOnUser struct {
	ID        uint   `gorm:"primaryKey;autoIncrement;->" json:"id"`
	ReportedUserID uuid.UUID `gorm:"type:uuid;not null;index" json:"reported"`
	ReportedUser User `gorm:"foreignKey:ReportedUserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	ReportCommon
}

type ReportOnPost struct {
	ID        uint   `gorm:"primaryKey;autoIncrement;->" json:"id"`
	ReportedPostID uint `gorm:"type:uuid;not null;index" json:"reported"`
	ReportedPost JobPost `gorm:"foreignKey:ReportedPostID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	ReportCommon
}

type ReportCommon struct {
	Reporter uuid.UUID `gorm:"type:uuid;not null;index" json:"reporter"`
	ReporterUser User `gorm:"foreignKey:Reporter;references:ID" json:"-"`

	Reason string `gorm:"type:text" json:"reason"`

	ReportTime int64 `gorm:"autoCreateTime;->" json:"report_time"`
	Status     string `gorm:"type:text;default:'pending';constraint:check(status in ('pending', 'resolved', 'rejected'))" json:"status"`

	AdminNote string `gorm:"type:text" json:"admin_note"`
}