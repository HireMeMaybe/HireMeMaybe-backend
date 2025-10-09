package model

import (
	"fmt"

	"github.com/google/uuid"
)

// Report status constants
var (
	ReportStatusPending  = "pending"
	ReportStatusResolved = "resolved"
	ReportStatusRejected = "rejected"
)

// UpdateableReport defines the interface for reports that can have their status updated.
type UpdateableReport interface {
	UpdateStatus(newStatus string, adminNote string) error
}

// ReportOnUser represents a report made against a user.
type ReportOnUser struct {
	ID             uint      `gorm:"primaryKey;autoIncrement;->" json:"id"`
	ReportedUserID uuid.UUID `gorm:"type:uuid;not null;index" json:"reported"`
	ReportedUser   User      `gorm:"foreignKey:ReportedUserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	ReportCommon
}

// ReportOnPost represents a report made against a job post.
type ReportOnPost struct {
	ID             uint    `gorm:"primaryKey;autoIncrement;->" json:"id"`
	ReportedPostID uint    `gorm:"type:uuid;not null;index" json:"reported"`
	ReportedPost   JobPost `gorm:"foreignKey:ReportedPostID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	ReportCommon
}

// ReportCommon contains fields common to all report types.
type ReportCommon struct {
	Reporter     uuid.UUID `gorm:"type:uuid;not null;index" json:"reporter"`
	ReporterUser User      `gorm:"foreignKey:Reporter;references:ID" json:"-"`

	Reason string `gorm:"type:text" json:"reason"`

	ReportTime int64  `gorm:"autoCreateTime;->" json:"report_time"`
	Status     string `gorm:"type:text;default:'pending';constraint:check(status in ('pending', 'resolved', 'rejected'))" json:"status"`

	AdminNote string `gorm:"type:text" json:"admin_note"`
}

// UpdateStatus updates the status and admin note of the report.
func (rc *ReportCommon) UpdateStatus(newStatus string, adminNote string) error {
	if newStatus != ReportStatusPending && newStatus != ReportStatusResolved && newStatus != ReportStatusRejected {
		return fmt.Errorf("invalid status: %s", newStatus)
	}

	rc.Status = newStatus
	rc.AdminNote = adminNote
	return nil
}
