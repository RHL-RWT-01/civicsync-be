package config

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	db     *mongo.Database
	client *mongo.Client
	once   sync.Once
)


// ConnectDB initializes and returns a MongoDB database connection
func ConnectDB() *mongo.Database {
	once.Do(func() {
		mongoURI := os.Getenv("MONGODB_URI")
		if mongoURI == "" {
			log.Fatal("Please define the MONGODB_URI environment variable")
		}

		// Create client
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		c, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
		if err != nil {
			log.Fatalf("Failed to create MongoDB client: %v", err)
		}

		if err := c.Connect(ctx); err != nil {
			log.Fatalf("Failed to connect to MongoDB: %v", err)
		}

		log.Println("Connected to MongoDB!")

		client = c
		db = client.Database("mydb")
	})

	return db
}

// GetCollection returns a MongoDB collection by name
func GetCollection(name string) *mongo.Collection {
	return ConnectDB().Collection(name)
}
