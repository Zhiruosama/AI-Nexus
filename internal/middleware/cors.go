// Package middleware CORS 中间件用于处理跨域问题
package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSconfig 定义CORS中间件的配置结构体
type CORSconfig struct {
	// AllowedOrigins 定义了允许访问资源的域列表，例如 ["http://localhost:3000", "https://app.example.com"]
	// 使用"*"表示允许所有域访问
	AllowedOrigins []string

	// AllowedMethods 定义了允许的HTTP方法
	// 例如["GET","POST","PUT"]
	AllowedMethods []string

	// AllowedHeaders 定义了允许在请求中携带的头部
	// 例如 ["Authorization", "Content-Type"]
	AllowedHeaders []string

	// ExposedHeaders 定义了允许浏览器访问的响应头
	ExposedHeaders []string

	// AllowCredentials 是否允许发送Cookie或HTTP认证信息
	AllowCredentials bool

	// MaxAge 是预检请求 (OPTIONS) 的缓存时间(秒)
	MaxAge time.Duration
}

// DefaultCORSConfig 返回一个默认配置
func DefaultCORSConfig() CORSconfig {
	return CORSconfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Origin", "Content-Length", "Content-Type", "Accept", "X-Request-ID", "Authorization"},
		ExposedHeaders:   []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// CORS 是处理跨域资源共享的中间件
func CORS(config CORSconfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求的源(Origin)
		origin := c.Request.Header.Get("Origin")

		// 检查Origin是否在允许列表内
		isAllowedOrigin := len(config.AllowedOrigins) == 0

		if !isAllowedOrigin && len(config.AllowedOrigins) > 0 {
			for _, allowed := range config.AllowedOrigins {
				if allowed == "*" || allowed == origin {
					isAllowedOrigin = true
					break
				}
			}
		}

		if isAllowedOrigin {
			// 设置 Access-Control-Allow-Origin
			// 注意：如果 AllowedOrigins 包含 "*", 则不能同时设置 AllowCredentials 为 true。
			// 通常将 * 替换为具体的 Origin。
			if len(config.AllowedOrigins) > 0 && config.AllowedOrigins[0] == "*" && config.AllowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(config.AllowedOrigins) > 0 && config.AllowedOrigins[0] == "*" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}

			//设置其他CORS响应头
			c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ","))
			c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ","))
			c.Writer.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))

			if config.AllowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Credetials", "true")
			}
		}
		c.Next()
	}
}
