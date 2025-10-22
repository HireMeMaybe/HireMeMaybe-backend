// Package model defines the data structures and relationships for user management in the application.
package model

import (
	"os"
	"strings"
	"time"

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

// Each punishment type
var (
	BanPunishment     = "ban"
	SuspendPunishment = "suspend"
)

// GoogleUserInfo struct holds the user information retrieved from Google OAuth	
type GoogleUserInfo struct {
	GID            string `json:"sub"`
	FirstName      string `json:"given_name"`
	LastName       string `json:"family_name"`
	Email          string `json:"email"`
	ProfilePicture string `json:"picture"`
}

// EditableUserInfo is part of User field that allow overwrite
type EditableUserInfo struct {
	Tel *string `json:"tel"`
}

// EditableCPSKInfo is part of CPSK field that allow overwrite
type EditableCPSKInfo struct {
	FirstName        string         `json:"first_name"`
	LastName         string         `json:"last_name"`
	Program          *string        `json:"program" gorm:"check:program IN ('CPE', 'SKE')"`
	EducationalLevel *string        `json:"year"`
	SoftSkill        pq.StringArray `gorm:"type:text[]" json:"soft_skill"`
}

// EditableCompanyInfo is part of company field that allow overwrite
type EditableCompanyInfo struct {
	Name     string  `json:"name"`
	Overview string  `json:"overview"`
	Industry string  `json:"industry"`
	Size     *string `json:"size" gorm:"check:size IN ('XS', 'S', 'M', 'L', 'XL')"`
}

// EditableVisitorInfo is part of visitor field that allow overwrite		
type EditableVisitorInfo struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// PunishmentStruct is for storing punishment detail like ban or suspend
type PunishmentStruct struct {
	ID             uint       `gorm:"primaryKey;autoIncrement;->" json:"-"`
	PunishmentType string     `json:"type"`
	PunishAt       *time.Time `json:"at"`
	PunishEnd      *time.Time `json:"end"`
}

// UserModel interface defines methods for user models
type UserModel interface {
	GetLoginResponse(string) interface{}
	GetID() uuid.UUID
	FillGoogleInfo(uInfo GoogleUserInfo)
}

// User struct is gorm model for store base user data in DB
type User struct {
	gorm.Model
	EditableUserInfo
	Email          *string           `json:"email" gorm:"<-:create"`
	ID             uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();primaryKey;<-:create" json:"id" `
	GoogleID       string            `json:"-" gorm:"<-:create"`
	Username       string            `json:"username" gorm:"<-:create"`
	Password       string            `json:"-"`
	Role           string            `json:"-"`
	PunishmentID   *int              `json:"-"`
	Punishment     *PunishmentStruct `json:"punishment"`
	ProfilePicture string            `json:"profile_picture"`
}

// GetID returns the user's UUID
func (u *User) GetID() uuid.UUID {
	return u.ID
}

// FillGoogleInfo fills the user struct with Google user info and assigns the role
func (u *User) FillGoogleInfo(uInfo GoogleUserInfo, role string) {
	u.Email = &uInfo.Email
	u.GoogleID = uInfo.GID
	u.Username = uInfo.FirstName
	u.ProfilePicture = uInfo.ProfilePicture
	u.Role = role
}

// CPSKUser is gorm model for store CPSK student profile data in DB
type CPSKUser struct {
	UserID uuid.UUID `json:"id" gorm:"primaryKey;<-:create"`
	User   User
	EditableCPSKInfo
	ResumeID *int `json:"resume_id"`
	Resume   File `json:"-"`

	// List of job applications made by the CPSK user
	Applications []Application `gorm:"foreignKey:CPSKID" json:"applications"`
}

// GetLoginResponse constructs the login response for CPSK user
func (c *CPSKUser) GetLoginResponse(accessToken string) interface{} {
	return CPSKResponse{User: *c, AccessToken: accessToken}
}

// GetID returns the CPSK user's UUID
func (c *CPSKUser) GetID() uuid.UUID {
	return c.User.GetID()
}

// FillGoogleInfo fills the CPSK user struct with Google user info
func (c *CPSKUser) FillGoogleInfo(uInfo GoogleUserInfo) {
	c.User.FillGoogleInfo(uInfo, RoleCPSK)
	c.FirstName = uInfo.FirstName
	c.LastName = uInfo.LastName
}

// CompanyUser is gorm model for store company relate data in DB
type CompanyUser struct {
	UserID         uuid.UUID `json:"id" gorm:"primaryKey;<-:create"`
	User           User
	VerifiedStatus string `json:"verified_status" gorm:"check:verified_status IN ('Pending', 'Verified', 'Unverified')"`
	EditableCompanyInfo
	LogoID   *int `json:"logo_id"`
	Logo     File `json:"-"`
	BannerID *int `json:"banner_id"`
	Banner   File `json:"-"`

	// JobPost holds the company's job posts
	JobPost []JobPost `gorm:"foreignKey:CompanyUserID" json:"job_post"`
}

// GetLoginResponse constructs the login response for Company user
func (c *CompanyUser) GetLoginResponse(accessToken string) interface{} {
	return &CompanyResponse{User: *c, AccessToken: accessToken}
}

// GetID returns the Company user's UUID
func (c *CompanyUser) GetID() uuid.UUID {
	return c.User.GetID()
}

// FillGoogleInfo fills the Company user struct with Google user info and sets verification status
func (c *CompanyUser) FillGoogleInfo(uInfo GoogleUserInfo) {

	verified := StatusPending
	if strings.ToLower(strings.TrimSpace(os.Getenv("BYPASS_VERIFICATION"))) == "true" {
		verified = StatusVerified
	}

	c.User.FillGoogleInfo(uInfo, RoleCompany)
	c.VerifiedStatus = verified
}

// VisitorUser is gorm model for store visitor relate data in DB
type VisitorUser struct {
	UserID uuid.UUID `json:"id" gorm:"primaryKey;<-:create"`
	User   User
	EditableVisitorInfo
}

// GetLoginResponse constructs the login response for Visitor user
func (v *VisitorUser) GetLoginResponse(accessToken string) interface{} {
	return &VisitorResponse{User: *v, AccessToken: accessToken}
}

// GetID returns the Visitor user's UUID
func (v *VisitorUser) GetID() uuid.UUID {
	return v.User.GetID()
}

// FillGoogleInfo fills the Visitor user struct with Google user info
func (v *VisitorUser) FillGoogleInfo(uInfo GoogleUserInfo) {
	v.User.FillGoogleInfo(uInfo, RoleVisitor)
	v.FirstName = uInfo.FirstName
	v.LastName = uInfo.LastName
}
