// Package middleware Logging 中间件的实现
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDKey
const RequestIDKey = "X-Request-ID"

// Logging负责为每个请求生成或提取唯一的 Request ID
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestID string
		// 优先从HTTP请求头里获取ID
		requestID = c.GetHeader(RequestIDKey)
		// 如果没有 则生成一个唯一ID
		if requestID == "" {
			requestID = uuid.New().String()
			// 将ID设置回响应头 客户端可以收到此ID用于报告问题跟追踪
			c.Writer.Header().Set(RequestIDKey, requestID)
		}
		// 将ID附加到当前的请求上下文中 供外部使用
		c.Set(RequestIDKey, requestID)

		//继续处理请求链
		c.Next()
	}
}

// 通过requestid追踪日志
