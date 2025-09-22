package main

import (
	"log"
	"net/http"

	"civicsync-be/config"
	"civicsync-be/routes"

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

	log.Println("MongoDB connection established successfully!")

	r := gin.Default()

	routes.UserRoutes(r)
	
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
