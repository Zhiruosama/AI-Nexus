// Package middleware 重复请求判断中间件
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
)

// DeduplicationMiddleware 中间件具体实现
func DeduplicationMiddleware() gin.HandlerFunc {
	rdbClient := rdb.Rdb
	ctx := rdb.Ctx
	lockDuration := configs.GlobalConfig.Deduplication.LockDuration

	if rdbClient == nil {
		panic("AntiReplayMiddleware requires Redis client (rdb.Rdb) to be initialized.")
	}

	return func(c *gin.Context) {
		method := c.Request.Method

		if method != http.MethodGet && method != http.MethodPut && method != http.MethodDelete && method != http.MethodPost {
			c.Abort()
			return
		}

		var userIdentifier string
		if userID, exists := c.Get(UserIDKey); exists {
			userIdentifier = fmt.Sprintf("user:%d", userID)
		} else {
			userIdentifier = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		// 路径和查询参数
		requestPath := c.Request.URL.Path + "?" + c.Request.URL.RawQuery

		// 处理请求体并计算哈希值
		bodyHash := ""
		if method == http.MethodPut || method == http.MethodPost {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "can't read request body"})
				return
			}

			h := sha256.New()
			h.Write(bodyBytes)
			bodyHash = hex.EncodeToString(h.Sum(nil))

			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 构建最终的 Redis 密钥
		// replay:<User/IP>:<Method>:<Path>:<BodyHash>
		redisKey := fmt.Sprintf("replay:%s:%s:%s:%s", userIdentifier, method, requestPath, bodyHash)

		// 尝试加锁
		wasSet, err := rdbClient.SetNX(ctx, redisKey, "locked", lockDuration).Result()

		if err != nil {
			fmt.Printf("Redis Deduplication check failed: %v. Request allowed.\n", err)
			c.Abort()
			return
		}

		if !wasSet {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error":   "Duplicate Request",
				"message": fmt.Sprintf("This exact request was already processed within the last %d seconds.", int(lockDuration.Seconds())),
			})
			return
		}

		c.Next()
	}
}
