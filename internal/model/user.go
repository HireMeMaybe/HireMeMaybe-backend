package model

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Tel   *string `json:"tel"`
	Email *string `json:"email"`
	ID       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	GoogleId string    `json:"-"`
	Username string    `json:"username"`
}

type CPSKUser struct {
	UserID    uuid.UUID
	User      User
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	Program   *string        `check:"year IN ('CPE', 'SKE')" json:"program"`
	Year      *string        `check:"year IN ('1', '2', '3', '4', 'Graduated')" json:"year"`
	SoftSkill pq.StringArray `gorm:"type:text[]" json:"soft_skill"`
}
