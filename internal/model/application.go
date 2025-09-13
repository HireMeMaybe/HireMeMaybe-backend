package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	// ApplicationStatusPending indicates that the application is pending review
	ApplicationStatusPending = "pending"
	// ApplicationStatusInConsideration indicates that the application is in consideration and company will contact later
	ApplicationStatusInConsideration = "in consideration"
	// ApplicationStatusRejected indicates that the application has been rejected
	ApplicationStatusRejected = "rejected"
)

// Application represents a job application record
type Application struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AppliedAt time.Time `gorm:"type:timestamp" json:"applied_at"`
	Status    string    `gorm:"type:text" json:"status"`

	// CPSKID references CPSKUser.UserID (uuid)
	CPSKID   uuid.UUID `gorm:"type:uuid;not null;index" json:"cpsk_id"`
	CPSKUser CPSKUser  `gorm:"foreignKey:CPSKID;references:UserID" json:"-"`

	// PostID references JobPost.ID
	PostID  uint    `gorm:"not null;index" json:"post_id"`
	JobPost JobPost `gorm:"foreignKey:PostID;references:ID" json:"-"`

	AnswerID uint             `json:"answer_id"`
	Answer   AplicationAnswer `gorm:"foreignKey:AnswerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"answer"`

	ResumeID *int `json:"resume_id"`
	Resume   File `gorm:"foreignKey:ResumeID;references:ID" json:"-"`
}

// AplicationAnswer represents additional answer for a job application
type AplicationAnswer struct {
	ID                   uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	RightToWork          string         `json:"right_to_work"`
	ExpectedSalary       string         `json:"expected_salary"`
	YearOfExperience     uint           `json:"year_of_experience"`
	ProgrammingLanguages pq.StringArray `json:"programming_languages" gorm:"type:text[]"`
}
