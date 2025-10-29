// Package middleware RequestID 中间件的实现
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDKey 响应头
const RequestIDKey = "X-RequeRst-ID"

// RequestID 负责为每个请求生成唯一的 Request ID
func RequestID() gin.HandlerFunc {

	return func(c *gin.Context) {
		var requestID = uuid.New().String()
		c.Set(RequestIDKey, requestID)
		c.Next()
	}
}
