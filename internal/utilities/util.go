// Package utilities contain utility code that use across the package
package utilities

import (
	"HireMeMaybe-backend/internal/model"
	"log"
	"net/http"
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

// ExtractUser will extract user model from gin context and abort with error message
func ExtractUser(c *gin.Context) model.User {
	u, _ := c.Get("user")
	if u == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "User information not provided",
		})
		return model.User{}
	}

	user, ok := u.(model.User)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to assert type",
		})
		return model.User{}
	}
	return user
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
