package middleware

import (
	"HireMeMaybe-backend/internal/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CheckRole(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := util.ExtractUser(ctx)

		if user.Role != role {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "User doesn't have permission to access",
			})
		}
	}
}