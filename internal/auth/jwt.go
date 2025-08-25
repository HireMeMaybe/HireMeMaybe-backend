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

var SECRET_KEY = os.Getenv("SECRET_KEY")

// TODO: generate refresh token
func generateToken(uuid uuid.UUID) (string, string, error) {

	generatedAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "HireMeMaybe",
		Subject: uuid.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	})

	signedToken, err := generatedAccessToken.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", "", fmt.Errorf("Failed to sign token: %s", err)
	}

	return signedToken, "", nil
}

func ValidatedToken(encodeToken string) (*jwt.Token, error) {
	return jwt.Parse(encodeToken, func(token *jwt.Token) (interface{}, error) {
		if _, isvalid := token.Method.(*jwt.SigningMethodHMAC); !isvalid {
			return nil, fmt.Errorf("Invalid token")

		}
		return []byte(SECRET_KEY), nil
	})
}