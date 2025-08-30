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

var secretKey = os.Getenv("SECRET_KEY")

// TODO: generate refresh token
func generateToken(uuid uuid.UUID) (string, string, error) {

	generatedAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "HireMeMaybe",
		Subject:   uuid.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})

	signedToken, err := generatedAccessToken.SignedString([]byte(secretKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, "", nil
}

// ValidatedToken parses and validates a JWT token using a secret key.
func ValidatedToken(encodeToken string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(encodeToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid token")

		}
		return []byte(secretKey), nil
	})
}
