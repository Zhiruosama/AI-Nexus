// Package middleware Idempotency 幂等设计中间件
package middleware

import (
	"fmt"
	"net/http"

	"github.com/Zhiruosama/ai_nexus/configs"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	"github.com/gin-gonic/gin"
)

func IdempotencyMiddleware() gin.HandlerFunc {
	//获取Redis客户端跟context
	rdbClient := rdb.Rdb
	ctx := rdb.Ctx
	lockDuration := configs.GlobalConfig.Idempotency.LockDuration

	if rdbClient == nil {
		panic("IdempotencyMiddleware requires Redis client (rdb.Rdb) to be initialized.")
	}

	return func(c *gin.Context) {
		// 仅对非幂等方法保护
		method := c.Request.Method
		if method != http.MethodPost {
			c.Next()
			return
		}

		// 获取幂等键
		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "POST/PUT 请求必须提供 Idempotency-Key 请求头"})
			return
		}

		// 构建 Redis Key
		var redisKey string
		if userID, exists := c.Get("user_id"); exists {
			// Key 包含用户ID (由 JWT 中间件注入) 以防止不同用户使用相同的 Key
			redisKey = fmt.Sprintf("id:user:%d:%s", userID, idempotencyKey)
		} else {
			// 没有用户ID 就用IP
			redisKey = fmt.Sprintf("id:ip:%s:%s", c.ClientIP(), idempotencyKey)
		}

		// 尝试加锁(SETNX)
		setResult, err := rdbClient.SetNX(ctx, redisKey, "processing", lockDuration).Result()
		if err != nil {
			fmt.Printf("Redis SETNX 失败: %v. 幂等性检查跳过\n", err)
			c.Next()
			return
		}

		// 判断是否重复请求
		if !setResult {
			// SETNX 返回 false 表示Key已经存在
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error":   "重复请求",
				"message": "该 Idempotency-Key 已被使用或正在处理",
			})
			return
		}

		//请求成功加锁，在请求结束时解锁
		defer func() {
			// 确保业务逻辑完成后释放锁，防止 Key 占用 LockDuration 那么久
			rdbClient.Del(ctx, redisKey)
		}()

		c.Next()
	}
}
