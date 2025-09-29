package config

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()
var RedisClient *redis.Client

// ConnectRedis initializes the Redis client
func ConnectRedis() {
	var redisAddr = os.Getenv("REDIS_ADDRESS") // change if using Docker or cloud
	var redisPassword = os.Getenv("REDIS_PASSWORD") // set if needed
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0, // default DB
	})

	_, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	fmt.Println("Connected to Redis")
}
