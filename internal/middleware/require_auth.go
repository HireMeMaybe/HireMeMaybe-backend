// Package middleware contain utilities middleware code
package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"errors"
	"HireMeMaybe-backend/internal/utilities"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

// RequireAuth function is a middleware in Go that validates a Bearer token in the Authorization
// header and checks if the user associated with the token exists and is not expired before allowing
// access to the endpoint.
func RequireAuth(db *database.DBinstanceStruct) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		const BearerSchema = "Bearer "
		authHeader := ctx.GetHeader("Authorization")

		if len(authHeader) <= len(BearerSchema) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, utilities.ErrorResponse{
				Error: "Invalid authorization header",
			})
			return
		}

		tokenString := authHeader[len(BearerSchema):]
		token, err := auth.ValidatedToken(tokenString)

		
		if !token.Valid {
			if errors.Is(err, jwt.ErrTokenExpired) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Access token expired",
				})
				return
			}

			if errors.Is(err, jwt.ErrTokenInvalidIssuer) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid token issuer",
				})
				return
			}

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": fmt.Sprintf("Failed to validate token: %s", err.Error()),
			})
			return
		}

		claims := token.Claims.(*jwt.RegisteredClaims)

		if claims.Issuer != auth.JwtIssuer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token issuer",
			})
			return
		}

		userID := claims.Subject

		var foundUser model.User

		if err := db.Where("id = ?", userID).First(&foundUser).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "User not exist",
				})
				return
			}

			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

		ctx.Set("user", foundUser)
		ctx.Next()
	}
}
