package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func RequireAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		const BEARER_SCHEMA = "Bearer "
		authHeader := ctx.GetHeader("Authorization")
		tokenString := authHeader[len(BEARER_SCHEMA):]
		token, err := auth.ValidatedToken(tokenString)

		if !token.Valid {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": fmt.Sprintf("Failed to validate token: %s", err.Error()),
			})
		}

		claims := token.Claims.(jwt.RegisteredClaims)

		if claims.ExpiresAt.Time.Unix() < time.Now().Unix() {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": "Access token expired",
			})
		}
	}
}