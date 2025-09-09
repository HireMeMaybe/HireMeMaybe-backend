package middleware

import (
	"HireMeMaybe-backend/internal/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckRole will protect endpoint from user that is not a specific roles
func CheckRole(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := utilities.ExtractUser(ctx)

		if user.Role != role {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "User doesn't have permission to access",
			})
		}
	}
}
