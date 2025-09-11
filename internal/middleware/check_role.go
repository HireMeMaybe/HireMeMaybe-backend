package middleware

import (
	"HireMeMaybe-backend/internal/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckRole will protect endpoint from user that is not a specific roles
func CheckRole(roles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := utilities.ExtractUser(ctx)

		if !contains(roles, user.Role) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "User doesn't have permission to access",
			})
		}
	}
}

func contains(slice []string, s string) bool {
    for _, v := range slice {
        if v == s {
            return true
        }
    }
    return false
}