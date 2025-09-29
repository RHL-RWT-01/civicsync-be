package middlewares

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"civicsync-be/config"
)

func IssueRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("user_id")
		
		key := "rate_limit:" + userID
		ctx := config.Ctx

		// Increment request count
		count, err := config.RedisClient.Incr(ctx, key).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error"})
			c.Abort()
			return
		}

		// If first request, set TTL
		if count == 1 {
			config.RedisClient.Expire(ctx, key, window)
		}

		// Block if exceeded limit
		if count > int64(limit) {
			ttl, _ := config.RedisClient.TTL(ctx, key).Result()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": ttl.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
