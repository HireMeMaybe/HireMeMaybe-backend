package model

import (
	"fmt"

	"github.com/google/uuid"
)

var (
	ReportStatusPending   = "pending"
	ReportStatusResolved  = "resolved"
	ReportStatusRejected  = "rejected"
)

type UpdateableReport interface {
	UpdateStatus(newStatus string, adminNote string) error
}

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

func (rc *ReportCommon) UpdateStatus(newStatus string, adminNote string) error {
	if newStatus != ReportStatusPending && newStatus != ReportStatusResolved && newStatus != ReportStatusRejected {
		return fmt.Errorf("invalid status: %s", newStatus)
	}

	rc.Status = newStatus
	rc.AdminNote = adminNote
	return nil
}