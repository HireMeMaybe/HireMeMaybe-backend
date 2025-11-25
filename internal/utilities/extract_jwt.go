package utilities

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func ExtractBearerToken(c *gin.Context) (string, error) {

	const BearerSchema = "Bearer "
	authHeader := c.GetHeader("Authorization")

	if len(authHeader) <= len(BearerSchema) {
		return "", fmt.Errorf("Invalid authorization header")
	}

	return authHeader[len(BearerSchema):], nil

}
