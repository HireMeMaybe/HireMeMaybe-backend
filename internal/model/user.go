package model

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type ContactInfo struct {
	Tel 	*string `json:"tel"`
	Email 	*string `json:"email"`
}

type CPSKUser struct {
	gorm.Model `gorm:"embedded"`
	ContactInfo `gorm:"embedded"`
	GoogleId 	string `json:"-"`
	FirstName 	string `json:"first_name"`
	LastName 	string `json:"last_name"`
	Program 	*string `check:"year IN ('CPE', 'SKE')" json:"program"` 
	Year 		*string `check:"year IN ('1', '2', '3', '4', 'Graduated')" json:"year"`
	SoftSkill 	pq.StringArray `gorm:"type:text[]" json:"soft_skill"`
}