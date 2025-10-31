// Package utils 密码加密工具
package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id参数配置
const (
	memory      = 64 * 1024 // 64MB
	iterations  = 3         // 迭代次数
	parallelism = 2         // 并行度
	saltLength  = 16        // 盐长度
	keyLength   = 32        // 密钥长度
)

// HashPassword 使用Argon2id加密密码
func HashPassword(password string) (string, error) {
	// 生成随机盐
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// 使用Argon2id生成哈希
	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)

	// 编码为base64格式存储
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// 格式: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, iterations, parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, encodedHash string) (bool, error) {
	// 解析存储的哈希
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	var version int
	var memory, iterations uint32
	var parallelism uint8

	// 解析参数
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, err
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, err
	}

	// 解码盐和哈希
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// 使用相同参数重新计算哈希
	comparisonHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(decodedHash)))

	// 使用constant time比较防止时序攻击
	return subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1, nil
}
