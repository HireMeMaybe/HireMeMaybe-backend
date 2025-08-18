// Package model contain gorm model for recording data to database
package model

import "gorm.io/gorm"

// SomeModel is model just for testing gorm
type SomeModel struct {
	gorm.Model `gorm:"embedded"`
	Name       string
}
