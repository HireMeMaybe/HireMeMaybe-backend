package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	// Load env file into environments.
	_ "github.com/joho/godotenv/autoload"
)

// TODO: generate refresh token
func generateToken(uuid uuid.UUID) (string, string, error) {
	generatedAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  uuid.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	signedToken, err := generatedAccessToken.SignedString([]byte(os.Getenv("SECRET_KEY")))
	if err != nil {
		return "", "", fmt.Errorf("Failed to sign token: %s", err)
	}

	return signedToken, "", nil
}
