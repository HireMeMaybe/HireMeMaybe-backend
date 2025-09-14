// Package utilities contain utility code that use across the package
package utilities

import (
	"HireMeMaybe-backend/internal/model"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ExtractUser will extract user model from gin context and abort with error message
func ExtractUser(c *gin.Context) model.User {
	u, _ := c.Get("user")
	if u == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "User information not provided",
		})
	}

	user, ok := u.(model.User)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to assert type",
		})
	}
	return user
}

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
