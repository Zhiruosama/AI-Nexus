// Package middleware deduplication 重复请求判断中间件
package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/Zhiruosama/ai_nexus/configs"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// 中间件具体实现
func DeduplicationMiddleware() gin.HandlerFunc {
	rdbClient := rdb.Rdb
	ctx := rdb.Ctx
	lockDuration := configs.GlobalConfig.Deduplication.LockDuration

	if rdbClient == nil {
		panic("AntiReplayMiddleware requires Redis client (rdb.Rdb) to be initialized.")
	}

	return func(c *gin.Context) {
		method := c.Request.Method

		// 仅对需要防重放的方法启用保护
		if method != http.MethodGet && method != http.MethodPut && method != http.MethodDelete {
			c.Next()
			return
		}

		// 获取用户标识 (已认证用户优先，否则使用IP)
		var userIdentifier string
		if userID, exists := c.Get("user_id"); exists {
			userIdentifier = fmt.Sprintf("user:%d", userID)
		} else {
			userIdentifier = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		// 路径和查询参数
		requestPath := c.Request.URL.Path + "?" + c.Request.URL.RawQuery

		// 处理请求体并计算哈希值
		bodyHash := ""
		if method == http.MethodPut {
			// 读取请求体，计算哈希，然后恢复请求体
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "无法读取请求体"})
				return
			}

			// 计算 SHA256 哈希
			h := sha256.New()
			h.Write(bodyBytes)
			bodyHash = hex.EncodeToString(h.Sum(nil))

			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 构建最终的 Redis 密钥
		// replay:<User/IP>:<Method>:<Path>:<BodyHash>
		redisKey := fmt.Sprintf("replay:%s:%s:%s:%s", userIdentifier, method, requestPath, bodyHash)

		// 尝试加锁
		setResult, err := rdbClient.Set(ctx, redisKey, "locked", lockDuration).Result()

		if err == redis.Nil {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "重复的请求",
				"message": fmt.Sprintf("该请求在%d秒内已处理，请勿重复提交", int(lockDuration.Seconds())),
			})
			return
		} else if err != nil {
			fmt.Printf("Redis Anti-Replay 失败: %v. 请求已放行\n", err)
			c.Next()
			return
		} else if setResult != "OK" {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "请求重放被拦截"})
			return
		}
		c.Next()
	}
}
