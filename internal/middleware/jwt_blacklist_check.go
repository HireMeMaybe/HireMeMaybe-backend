package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/utilities"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// JwtBlacklistCheck is a middleware that checks if the JWT token is blacklisted
func JwtBlacklistCheck(bl auth.JwtBlacklistStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString, err := utilities.ExtractBearerToken(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, utilities.ErrorResponse{
				Error: err.Error(),
			})
			return
		}

		isBlacklisted, err := bl.IsBlacklisted(tokenString)

		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to validate token: %s", err.Error()),
			})
			return
		}

		if isBlacklisted {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{
				Error: "Token has been revoked",
			})
			return
		}

		fmt.Println("i'm here =================================")
	}
}
