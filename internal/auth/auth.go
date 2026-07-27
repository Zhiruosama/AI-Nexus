// Package auth implements JWT issuance and revocable session authentication.
package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Zhiruosama/ai_nexus/internal/pkg/session"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	tokenIssuer = "ai-nexus-auth"
	tokenTTL    = 7 * 24 * time.Hour

	// UserIDContextKey keeps compatibility with existing handlers.
	UserIDContextKey = "user_id"
	// PrincipalContextKey stores the authenticated request identity.
	PrincipalContextKey = "principal"
)

var errJWTSecretNotSet = errors.New("JWT_SECRET not set")

// Claims contains standard JWT claims. Subject is the user ID and ID is the
// server-side session ID.
type Claims struct {
	jwt.RegisteredClaims
}

// GeneratedToken contains the signed token and its server-side session metadata.
type GeneratedToken struct {
	Value     string
	SessionID string
	ExpiresAt time.Time
}

// Principal is the authenticated identity for one request.
type Principal struct {
	UserID    string
	SessionID string
}

// GenerateToken signs a token for a new login session.
func GenerateToken(userID string) (*GeneratedToken, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	now := time.Now().UTC().Truncate(time.Second)
	expirationTime := now.Add(tokenTTL)
	sessionID := uuid.NewString()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    tokenIssuer,
			Subject:   userID,
			ID:        sessionID,
		},
	}

	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return nil, err
	}

	return &GeneratedToken{
		Value:     tokenString,
		SessionID: sessionID,
		ExpiresAt: expirationTime,
	}, nil
}

// ParseToken verifies the signature, algorithm, issuer and registered claims.
func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret()
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.Subject == "" || claims.ID == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// AuthenticateToken validates both the JWT and the server-side session.
func AuthenticateToken(ctx context.Context, tokenString string, sessionStore session.Store) (*Principal, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	userID, err := sessionStore.UserID(ctx, claims.ID)
	if errors.Is(err, session.ErrNotFound) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, &BackendError{cause: err}
	}
	if userID != claims.Subject {
		return nil, session.ErrNotFound
	}

	return &Principal{
		UserID:    claims.Subject,
		SessionID: claims.ID,
	}, nil
}

// Middleware authenticates a bearer token and stores its Principal in Gin.
func Middleware(sessionStore session.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := bearerToken(c.GetHeader("Authorization"))
		if err != nil {
			reject(c)
			return
		}

		principal, err := AuthenticateToken(c.Request.Context(), tokenString, sessionStore)
		if err != nil {
			var backendErr *BackendError
			if errors.As(err, &backendErr) {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error": "authentication service unavailable",
				})
				return
			}
			reject(c)
			return
		}

		c.Set(PrincipalContextKey, principal)
		c.Set(UserIDContextKey, principal.UserID)
		c.Next()
	}
}

// CurrentPrincipal returns the identity established by Middleware.
func CurrentPrincipal(c *gin.Context) (*Principal, bool) {
	value, ok := c.Get(PrincipalContextKey)
	if !ok {
		return nil, false
	}
	principal, ok := value.(*Principal)
	return principal, ok
}

// BackendError indicates that authentication failed because the session store
// was unavailable rather than because the credentials were invalid.
type BackendError struct {
	cause error
}

func (e *BackendError) Error() string {
	return "authentication backend: " + e.cause.Error()
}

func (e *BackendError) Unwrap() error {
	return e.cause
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("invalid bearer token")
	}
	return parts[1], nil
}

func reject(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "invalid or expired authentication",
	})
}

func jwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errJWTSecretNotSet
	}
	return []byte(secret), nil
}
