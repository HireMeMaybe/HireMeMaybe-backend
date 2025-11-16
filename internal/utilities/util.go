// Package utilities contain utility code that use across the package
package utilities

import (
	"HireMeMaybe-backend/internal/model"
	"errors"
	"log"
	"reflect"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ErrorResponse type for swagger docs
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse type for swagger docs
type MessageResponse struct {
	Message string `json:"message"`
}

// ExtractUser extracts the user model from Gin context.
// It no longer aborts the request; instead returns an error when missing/invalid.
func ExtractUser(c *gin.Context) (model.User, error) {
	u, _ := c.Get("user")
	if u == nil {
		return model.User{}, errors.New("User information not provided")
	}

	user, ok := u.(model.User)
	if !ok {
		return model.User{}, errors.New("Failed to assert type")
	}
	return user, nil
}

// CreateAdmin creates an admin user with the given password and username in the provided database.
func CreateAdmin(password string, username string, db *gorm.DB) {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		log.Fatal("failed to hash password: ", err)
	}

	// Create admin user
	admin := model.User{
		Username: username,
		Password: hashedPassword,
		Role:     model.RoleAdmin,
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Fatal("failed to create admin: ", err)
	}
}

// MergeNonEmpty help merge struct with non-empty field
func MergeNonEmpty(dst, src interface{}) {
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src).Elem()

	for i := 0; i < sv.NumField(); i++ {
		sf := sv.Field(i)
		if !sf.IsZero() {
			df := dv.FieldByName(sv.Type().Field(i).Name)
			if df.IsValid() && df.CanSet() {
				df.Set(sf)
			}
		}
	}
}
