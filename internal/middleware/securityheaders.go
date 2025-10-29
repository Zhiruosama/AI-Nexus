// Package middleware 添加安全用HTTP响应头中间件
package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders 安全头中间件核心逻辑实现
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防御 XSS
		// c.Writer.Header().Set("Content-Security-Policy", "default-src 'self';")

		// 拒绝在 Iframe 中嵌入页面
		c.Writer.Header().Set("X-Frame-Options", "DENY")

		// 防止MIME类型嗅探
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		c.Next()
	}
}
