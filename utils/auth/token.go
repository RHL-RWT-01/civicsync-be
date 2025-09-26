package authUtils

import (
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// GenerateAndSetToken generates a JWT token for a given user ID
func GenerateAndSetToken(userID string) (string, error) {
	secretStr := os.Getenv("JWT_SECRET")
	if secretStr == "" {
		return "", fmt.Errorf("JWT_SECRET environment variable is not set")
	}

	jwtSecret := []byte(secretStr)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Token expires in 72 hours
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
