package middlewares

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

// AuthMiddleware validates JWT tokens and protects routes
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from cookie
		tokenString, err := c.Cookie("auth_token")
		if err != nil || tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization token provided"})
			c.Abort()
			return
		}

		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "JWT secret not configured"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			log.Printf("Token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if userID, exists := claims["user_id"]; exists {
				c.Set("user_id", userID)
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Next()
	}
}
