// Package middleware Ratelimiting 流量控制中间件
package middleware

import (
	"fmt"
	"net/http"

	"github.com/Zhiruosama/ai_nexus/configs"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	"github.com/gin-gonic/gin"
)

// RateLimitingMiddleware 中间件具体实现
func RateLimitingMiddleware() gin.HandlerFunc {
	// 读取配置参数
	maxRequests := configs.GlobalConfig.RateLimit.LimitMax
	windowDuration := configs.GlobalConfig.RateLimit.Window

	// 获取Redis客户端实例
	rdbClient := rdb.Rdb
	ctx := rdb.Ctx

	// 检查是否成功获取到Redis
	if rdbClient == nil {
		panic("RateLimitingMiddleware requires Redis client (rdb.Rdb) to be initialized before use.")
	}

	return func(c *gin.Context) {
		// 获取用户标识
		var key string
		if userID, exists := c.Get("user_id"); exists {
			//JWT认证通过，使用用户ID作为key
			key = fmt.Sprintf("rl:user:%d", userID)
		} else {
			//未认证请求，使用客户端IP作为key
			key = fmt.Sprintf("rl:ip:%s", c.ClientIP())
		}

		// 计数(使用INCR)
		count, err := rdbClient.Incr(ctx, key).Result()
		if err != nil {
			//Redis故障处理
			fmt.Printf("Redis INCR 失败: %v. 请求已放行\n", err)
			c.Next()
			return
		}

		// 设置过期时间 且仅在第一次请求时设置
		if count == 1 {
			rdbClient.Expire(ctx, key, windowDuration)
		}

		// 判断是否超限
		if count > int64(maxRequests) {
			// 计算用户何时可以重试
			ttl := rdbClient.TTL(ctx, key).Val()
			retryAfter := int(ttl.Seconds())

			c.Writer.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "请求频次过高",
				"message": fmt.Sprintf("在 %d 秒内最多允许 %d 次请求，请在 %d 秒后重试。", int(windowDuration.Seconds()), maxRequests, retryAfter),
			})
			return
		}

		// 放行
		c.Next()
	}
}
