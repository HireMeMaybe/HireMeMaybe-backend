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
		if ctx.IsAborted() {
			return
		}
		if !utilities.Contains(roles, user.Role) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, utilities.ErrorResponse{
				Error: "User doesn't have permission to access",
			})
		}
	}
}
