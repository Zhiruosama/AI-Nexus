// Package middleware JWT工具包
package middleware

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 定义JWT密钥 从环境变量中加载
var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		panic("[ERROR] JWT_SECRET not set")
	}
	jwtSecret = []byte(secret)
}

// Claims 定义了 JWT Payload中包含的而信息
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken 用于生成 JWT Token
func GenerateToken(userID string) (string, error) {
	expirationTime := time.Now().Add(time.Hour * 24 * 7)

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ai-nexus-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)

	return tokenString, err
}

// ParseToken 用于解析和验证 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenExpired
	}

	return claims, nil
}
