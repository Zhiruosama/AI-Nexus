// Package middleware JWT 认证中间件。
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Zhiruosama/ai_nexus/internal/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware 创建一个用于验证 JWT Token 的 Gin 中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//从请求头里获取Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			//如果没有Authorization则拒绝访问
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "需要 Authorization 头'"})
			return
		}

		//验证Token格式
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			//格式错误
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization 格式无效，必须是 'Bearer <token>'"})
			return
		}

		tokenString := parts[1]

		//解析验证Token
		claims, err := util.ParseToken(tokenString)

		if err != nil {
			// Token验证失败

			// 检查是否过期
			if errors.Is(err, jwt.ErrTokenExpired) || strings.Contains(err.Error(), "expired") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token 已过期，请重新登录"})
				return
			}

			// 其他验证失败（如签名错误、格式错误等）
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token 无效或签名错误"})
			return
		}

		//验证成功 注入上下文
		c.Set("user_id", claims.UserID)

		//next
		c.Next()
	}
}
