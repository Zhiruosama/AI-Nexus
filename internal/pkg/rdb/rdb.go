// Redis初始化模块
package rdb

import (
	"context"
	"fmt"

	"github.com/Zhiruosama/ai_nexus/configs"
	"github.com/redis/go-redis/v9"
)

var (
	Rdb *redis.Client
	Ctx = context.Background()
)

func init() {
	cfg := configs.GlobalConfig.Redis

	Rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := Rdb.Ping(Ctx).Err(); err != nil {
		panic(fmt.Sprintf("[ERROR] Redis init error, err is: %s", err.Error()))
	}
	fmt.Println("Redis connect success")
}
