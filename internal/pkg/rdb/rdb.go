// Redis初始化模块
package rdb

import (
	"context"
	"log"

	"github.com/Zhiruosama/ai_nexus/configs"
	"github.com/redis/go-redis/v9"
)

var (
	Rdb *redis.Client
	Ctx = context.Background()
)

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
