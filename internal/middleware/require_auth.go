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

func RequireAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		const BEARER_SCHEMA = "Bearer "
		authHeader := ctx.GetHeader("Authorization")
		tokenString := authHeader[len(BEARER_SCHEMA):]
		token, err := auth.ValidatedToken(tokenString)

		if !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": fmt.Sprintf("Failed to validate token: %s", err.Error()),
			})
			return
		}

		claims := token.Claims.(*jwt.RegisteredClaims)
		fmt.Println(claims)

		
		if claims.ExpiresAt.Time.Before(time.Now()) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Access token expired",
			})
			return
		}

		userId := claims.Subject 
		
		var foundUser model.User

		if err := database.DBinstance.Where("id = ?", userId).First(&foundUser).Error; err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve user data",
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