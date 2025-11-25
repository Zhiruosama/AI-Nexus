// Package middleware JWT认证中间件
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// UserIDKey 用户 ID 上下文
const UserIDKey = "user_id"

// AuthMiddleware 创建一个用于验证 JWT Token 的 Gin 中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头里获取 Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 如果没有 Authorization 则拒绝访问
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "need Authorization in request head'"})
			return
		}

		// 验证Token格式
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization has invalid fmt, such as: 'Bearer <token>'"})
			return
		}

		tokenString := parts[1]

		// 解析验证Token
		claims, err := ParseToken(tokenString)

		if err != nil {
			if errors.Is(err, jwt.ErrSignatureInvalid) || errors.Is(err, jwt.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token error"})
				return
			}
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}
