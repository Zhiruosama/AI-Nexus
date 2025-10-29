// Package middleware CORS中间件 用于处理跨域问题
package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig 定义CORS中间件的配置结构体
type CORSConfig struct {
	// AllowedOrigins 定义了允许访问资源的域列表
	AllowedOrigins []string

	// AllowedMethods 定义了允许的HTTP方法
	AllowedMethods []string

	// AllowedHeaders 定义了允许在请求中携带的头部
	AllowedHeaders []string

	// ExposedHeaders 定义了允许浏览器访问的响应头
	ExposedHeaders []string

	// AllowCredentials 是否允许发送Cookie或HTTP认证信息
	AllowCredentials bool

	// MaxAge 是预检请求的缓存时间
	MaxAge time.Duration
}

// DefaultCORSConfig 返回一个默认配置
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Length", "Content-Type", "Accept", "X-Request-ID", "Authorization"},
		ExposedHeaders:   []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// CORS 是处理跨域资源共享的中间件
func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin == "" {
			c.Next()
			return
		}

		// 检查Origin是否在允许列表内
		isAllowedOrigin := false
		for _, allowed := range config.AllowedOrigins {
			if allowed == "*" || allowed == origin {
				isAllowedOrigin = true
				break
			}
		}

		if !isAllowedOrigin {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
		c.Header("Vary", "Origin")

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%.0f", config.MaxAge.Seconds()))
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
