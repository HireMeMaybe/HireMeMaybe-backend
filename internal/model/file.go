package model

type File struct {
	ID        int `gorm:"primaryKey"`
	Content   []byte
	Extension string
}
