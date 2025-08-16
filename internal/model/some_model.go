package model

import "gorm.io/gorm"

type SomeModel struct {
	gorm.Model `gorm:"embedded"`
	Name string
}


