package auth

import (
	"HireMeMaybe-backend/internal/utilities"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// LogoutController handles user logout by blacklisting JWT tokens
type LogoutController struct {
	BlacklistStore JwtBlacklistStore
}

// NewLogoutController creates a new instance of LogoutController
func NewLogoutController(blacklistStore JwtBlacklistStore) *LogoutController {
	return &LogoutController{
		BlacklistStore: blacklistStore,
	}
}

// LogoutHandler handles user logout by blacklisting the JWT token
func (lc *LogoutController) LogoutHandler(c *gin.Context) {

	tokenString, err := utilities.ExtractBearerToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{Error: err.Error()})
		return
	}

	claims, err := extractClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{Error: err.Error()})
		return
	}

	err = lc.BlacklistStore.AddToBlacklist(tokenString, claims.ExpiresAt.Time)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{Error: "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func extractClaims(c *gin.Context) (*jwt.RegisteredClaims, error) {
	claims, ok := c.Get("claims")
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	realClaims, okCast := claims.(*jwt.RegisteredClaims)
	if !okCast {
		return nil, fmt.Errorf("invalid token claims type")
	}
	return realClaims, nil
}
