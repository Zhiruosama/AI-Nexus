// Package rdb Redis初始化模块
package rdb

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"

	"github.com/Zhiruosama/ai_nexus/configs"
)

var (
	// Rdb Redis客户端实例
	Rdb *redis.Client
	// Ctx 上下文
	Ctx = context.Background()
)

func init() {
	cfg := configs.GlobalConfig.Redis

	Rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})

	if err := Rdb.Ping(Ctx).Err(); err != nil {
		panic(fmt.Sprintf("[ERROR] Redis init error, err is: %s", err.Error()))
	}
	log.Println("[Redis] Redis connect success")
}
