package utilities

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// ExtractBearerToken extracts the JWT token from the Authorization header in the format
func ExtractBearerToken(c *gin.Context) (string, error) {

	const BearerSchema = "Bearer "
	authHeader := c.GetHeader("Authorization")

	if len(authHeader) <= len(BearerSchema) {
		return "", fmt.Errorf("invalid authorization header")
	}

	return authHeader[len(BearerSchema):], nil

}
