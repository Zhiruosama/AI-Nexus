// Package middleware 流量控制中间件
package middleware

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Zhiruosama/ai_nexus/configs"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitingMiddleware 中间件具体实现
func RateLimitingMiddleware() gin.HandlerFunc {
	luaScript, err := os.ReadFile(filepath.Join("middleware", "lua", "rate_limit.lua"))
	if err != nil {
		panic(fmt.Sprintf("Failed to read rate limit lua script: %v", err))
	}
	script := redis.NewScript(string(luaScript))

	maxRequests := configs.GlobalConfig.RateLimit.LimitMax
	windowDuration := configs.GlobalConfig.RateLimit.Window
	windowSeconds := int(windowDuration.Seconds())

	rdbClient := rdb.Rdb
	ctx := rdb.Ctx

	return func(c *gin.Context) {
		var key string
		if userID, exists := c.Get(UserIDKey); exists {
			key = fmt.Sprintf("rl:user:%v", userID)
		} else {
			key = fmt.Sprintf("rl:ip:%s", c.ClientIP())
		}

		if key == "" {
			key = "default_ratelimit_key"
		}

		result, err := script.Run(ctx, rdbClient, []string{key}, windowSeconds).Result()
		if err != nil {
			fmt.Printf("Redis script execution failed: %v. Request allowed.\n", err)
			c.Next()
			return
		}

		count := result.(int64)

		// 判断是否超限
		if count > int64(maxRequests) {
			// 计算用户何时可以重试
			ttl := rdbClient.TTL(ctx, key).Val()
			retryAfter := int(ttl.Seconds())

			if retryAfter <= 0 {
				retryAfter = windowSeconds
			}

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "request too fast",
				"message": fmt.Sprintf("Please try again after %d seconds", retryAfter),
			})
			return
		}

		c.Next()
	}
}
