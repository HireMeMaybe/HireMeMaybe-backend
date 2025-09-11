// Package middleware contain utilities middleware code
package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// RequireAuth function is a middleware in Go that validates a Bearer token in the Authorization
// header and checks if the user associated with the token exists and is not expired before allowing
// access to the endpoint.
func RequireAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		const BearerSchema = "Bearer "
		authHeader := ctx.GetHeader("Authorization")

		if len(authHeader) <= len(BearerSchema) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Invalid authorization header",
			})
			return
		}

		tokenString := authHeader[len(BearerSchema):]
		token, err := auth.ValidatedToken(tokenString)

		if !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": fmt.Sprintf("Failed to validate token: %s", err.Error()),
			})
			return
		}

		claims := token.Claims.(*jwt.RegisteredClaims)

		if claims.ExpiresAt.Before(time.Now()) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Access token expired",
			})
			return
		}

		userID := claims.Subject

		var foundUser model.User

		if err := database.DBinstance.Where("id = ?", userID).First(&foundUser).Error; err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

		var defaultUUID uuid.UUID

		if foundUser.ID.String() == defaultUUID.String() {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "User not exist",
			})
			return
		}

		ctx.Set("user", foundUser)
		ctx.Next()
	}
}
