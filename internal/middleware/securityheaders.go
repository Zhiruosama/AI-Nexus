// 添加安全用HTTP响应头中间件
package middleware

import "github.com/gin-gonic/gin"

// 中间件核心逻辑实现
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止MIME类型嗅探
		// 告诉浏览器不要猜测响应的MIME类型 必须使用 Content-Type 指定的类型
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		// XSS保护
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		// 拒绝在 Iframe 中嵌入页面 (点击劫持防护 - Clickjacking)
		// 阻止其他网站将您的页面嵌入到 <frame>, <iframe> 或 <object> 中
		c.Writer.Header().Set("X-Frame-Options", "DENY")

		// 继续执行后续Handler
		c.Next()
	}
}
