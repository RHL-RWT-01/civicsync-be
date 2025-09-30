package main

import (
	"civicsync-be/config"
	"civicsync-be/routes"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	db := config.ConnectDB()
	if db == nil {
		log.Fatal("Failed to connect to MongoDB")
	}
	config.ConnectRedis()

	log.Println("MongoDB connection established successfully!")

	r := gin.Default()
	var clientURL = os.Getenv("CLIENT_URL")
	fmt.Println("Client URL:", clientURL)
	// Use CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{clientURL}, // frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	routes.AuthRoutes(r)
	routes.IssueRoutes(r)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
