// Package model defines the data structures and relationships for user management in the application.
package model

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Each user role string
var (
	RoleAdmin   = "admin"
	RoleCPSK    = "cpsk"
	RoleCompany = "company"
	RoleVisitor = "visitor"
)

// Each status for company
var (
	StatusPending    = "Pending"
	StatusVerified   = "Verified"
	StatusUnverified = "Unverified"
)

// Company field that allow overwrite
type EditableCompanyInfo struct {
	Name     string  `json:"name"`
	Overview string  `json:"overview"`
	Industry string  `json:"industry"`
	Size     *string `json:"size" gorm:"check:size IN ('XS', 'S', 'M', 'L', 'XL')"`
}

// User struct is gorm model for store base user data in DB
type User struct {
	gorm.Model
	Tel            *string   `json:"tel"`
	Email          *string   `json:"email" gorm:"<-:create"`
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;<-:create" json:"id" `
	GoogleID       string    `json:"-" gorm:"<-:create"`
	Username       string    `json:"username" gorm:"<-:create"`
	Password       string    `json:"-"`
	Role           string    `json:"-"`
	ProfilePicture string    `json:"profile_picture"`
}

// CPSKUser is gorm model for store CPSK student profile data in DB
type CPSKUser struct {
	UserID           uuid.UUID `json:"id" gorm:"primaryKey;<-:create"`
	User             User
	FirstName        string         `json:"first_name"`
	LastName         string         `json:"last_name"`
	Program          *string        `json:"program" gorm:"check:program IN ('CPE', 'SKE')"`
	EducationalLevel *string        `json:"year"`
	SoftSkill        pq.StringArray `gorm:"type:text[]" json:"soft_skill"`
	ResumeID         *int           `json:"resume_id"`
	Resume           File           `json:"-"`

	// List of job applications made by the CPSK user
	Applications []Application `gorm:"foreignKey:CPSKID" json:"applications"`
}

// Company is gorm model for store company relate data in DB
type Company struct {
	UserID         uuid.UUID `json:"id" gorm:"primaryKey;<-:create"`
	User           User
	VerifiedStatus string `json:"verified_status" gorm:"check:verified_status IN ('Pending', 'Verified', 'Unverified')"`
	EditableCompanyInfo
	LogoID   *int `json:"logo_id"`
	Logo     File `json:"-"`
	BannerID *int `json:"banner_id"`
	Banner   File `json:"-"`

	// JobPost holds the company's job posts
	JobPost []JobPost `gorm:"foreignKey:CompanyID" json:"job_post"`
}
