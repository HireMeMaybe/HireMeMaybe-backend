// Package middleware contain utilities middleware code
package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
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
		tokenString, err := utilities.ExtractBearerToken(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, utilities.ErrorResponse{
				Error: err.Error(),
			})
			return
		}

		token, err := auth.ValidatedToken(tokenString)

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{
					Error: "Access token expired",
				})
				return
			}

			if errors.Is(err, jwt.ErrTokenInvalidIssuer) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{
					Error: "Invalid token issuer",
				})
				return
			}

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to validate token: %s", err.Error()),
			})
			return
		}

		if !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{
				Error: "Invalid access token",
			})
			return
		}

		claims := token.Claims.(*jwt.RegisteredClaims)
		ctx.Set("claims", claims)

		if claims.Issuer != auth.JwtIssuer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{
				Error: "Invalid token issuer",
			})
			return
		}

		userID := claims.Subject

		var foundUser model.User

		if err := db.Preload("Punishment").Where("id = ?", userID).First(&foundUser).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{
					Error: "User not exist",
				})
				return
			}

			ctx.AbortWithStatusJSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

		ctx.Set("user", foundUser)
		ctx.Next()
	}
}
