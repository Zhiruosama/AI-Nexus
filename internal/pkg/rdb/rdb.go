// Package rdb Redis初始化模块
package rdb

import (
	"context"
	"log"

	"github.com/Zhiruosama/ai_nexus/configs"
	"github.com/redis/go-redis/v9"
)

var (
	// Rdb 是用于管理 Redis 数据库连接的客户端实例。
	Rdb *redis.Client
	// Ctx 是一个空的根 Context 用于需要默认 Context 的地方
	Ctx = context.Background()
)

// Init 对数据库实例进行初始化
func Init() {
	cfg := configs.GlobalConfig.Redis

	Rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := Rdb.Ping(Ctx).Err(); err != nil {
		log.Fatalf("Redis连接失败: %v", err)
	}
	log.Println("Redis连接成功")
}
