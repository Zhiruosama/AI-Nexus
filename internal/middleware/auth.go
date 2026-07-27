// Package middleware JWT认证中间件
package middleware

import (
	"context"

	internalauth "github.com/Zhiruosama/ai_nexus/internal/auth"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/session"
	"github.com/gin-gonic/gin"
)

const (
	// UserIDKey keeps compatibility with existing handlers.
	UserIDKey = internalauth.UserIDContextKey
	// PrincipalKey stores the authenticated request identity.
	PrincipalKey = internalauth.PrincipalContextKey
)

// Principal is the authenticated identity for one request.
type Principal = internalauth.Principal

// AuthMiddleware 创建一个用于验证 JWT Token 的 Gin 中间件
func AuthMiddleware() gin.HandlerFunc {
	return AuthMiddlewareWithStore(session.NewRedisStore(rdb.Rdb))
}

// AuthMiddlewareWithStore creates an authentication middleware with an
// injectable session store, allowing the security rules to be tested.
func AuthMiddlewareWithStore(sessionStore session.Store) gin.HandlerFunc {
	return internalauth.Middleware(sessionStore)
}

// AuthenticateToken validates both the JWT and the server-side session.
func AuthenticateToken(ctx context.Context, tokenString string, sessionStore session.Store) (*Principal, error) {
	return internalauth.AuthenticateToken(ctx, tokenString, sessionStore)
}

// CurrentPrincipal returns the identity established by AuthMiddleware.
func CurrentPrincipal(c *gin.Context) (*Principal, bool) {
	return internalauth.CurrentPrincipal(c)
}
