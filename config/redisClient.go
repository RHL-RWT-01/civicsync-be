package config

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()
var RedisClient *redis.Client

// ConnectRedis initializes the Redis client
func ConnectRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // change if using Docker or cloud
		Password: "",               // no password by default
		DB:       0,                // default DB
	})

	_, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	fmt.Println("✅ Connected to Redis")
}
