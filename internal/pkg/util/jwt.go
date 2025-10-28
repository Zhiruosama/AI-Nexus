package util

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 定义JWT密钥 从环境变量中加载)
var jwtSecret []byte

// init() 函数在包被导入时执行
func init() {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		fmt.Println("[ERROR]环境变量JWT_SECRET未设置")
	} else {
		jwtSecret = []byte(secret)
	}
}

// Claims 定义了 JWT Payload中包含的而信息
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// getSecretKey 返回JWT密钥
func getSecretKey() []byte {
	return jwtSecret
}

// GenerateToken用于生成JWT Token
// 参数 userID 要包含在Token中的用户ID
func GenerateToken(userID uint) (string, error) {
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
	// 使用getSecretKey获取密钥
	tokenString, err := token.SignedString(getSecretKey())

	return tokenString, err
}

// ParseToken 用于解析和验证 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	//解析Token 同时验证签名
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 确保签名方法是HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil //返回密钥 用于验证签名
	})

	// 检查是否有解析错误
	if err != nil {
		return nil, err
	}

	// 检查Token是否过期
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}

	return claims, nil
}
