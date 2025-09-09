// Package util contain utility code that use across the package
package util

import (
	"HireMeMaybe-backend/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
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
