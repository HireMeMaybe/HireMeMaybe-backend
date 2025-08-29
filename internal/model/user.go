package model

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Tel      *string   `json:"tel"`
	Email    *string   `json:"email" gorm:"<-:create"`
	ID       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;<-:create" json:"id" `
	GoogleId string    `json:"-" gorm:"<-:create"`
	Username string    `json:"username" gorm:"<-:create"`
}

type CPSKUser struct {
	UserID           uuid.UUID `json:"id" binding:"required" gorm:"primaryKey;<-:create"`
	User             User
	FirstName        string         `json:"first_name"`
	LastName         string         `json:"last_name"`
	Program          *string        `check:"year IN ('CPE', 'SKE')" json:"program"`
	EducationalLevel *string        `json:"year"`
	SoftSkill        pq.StringArray `gorm:"type:text[]" json:"soft_skill"`
	ResumeID         int
	Resume           File
}

type Company struct {
	UserID         uuid.UUID
	User           User
	VerifiedStatus string `check:"verified_status IN ('Pending', 'Verified', 'Unverified')" json:"verified_status"`
	Name           string `json:"name"`
	Overview       string `json:"overview"`
	Industry       string `json:"industry"`
	Size           string `json:"size"`
}
