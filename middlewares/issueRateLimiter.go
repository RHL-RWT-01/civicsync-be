package middlewares

import (
	"net/http"
	"os"
	"time"

	"civicsync-be/config"

	"github.com/gin-gonic/gin"
)

func IssueRateLimiter(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, _ := c.Get("user_id")
		userID, ok := userIDVal.(string)
		if !ok || userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id cookie missing"})
			c.Abort()
			return
		}

		ctx := config.Ctx
		queuePrefix := os.Getenv("REDIS_QUEUE_FOR_ISSUE_LIMIT")
		if queuePrefix == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis queue not configured"})
			c.Abort()
			return
		}

		// Create individual key for each user
		userKey := queuePrefix + ":" + userID

		// Increment user's count with TTL
		count, err := config.RedisClient.Incr(ctx, userKey).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error incrementing count"})
			c.Abort()
			return
		}

		// Set TTL only for the first increment (when count = 1)
		if count == 1 {
			err = config.RedisClient.Expire(ctx, userKey, 24*time.Hour).Err()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error setting TTL"})
				c.Abort()
				return
			}
		}

		// Check if user exceeded limit
		if count > int64(limit) {
			retryAfter, _ := config.RedisClient.TTL(ctx, userKey).Result()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
