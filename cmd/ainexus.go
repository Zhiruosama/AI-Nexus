// Package main 应用的入口点，初始化 Config 和 DB 配置
package main

import (
	_ "github.com/Zhiruosama/ai_nexus/configs"
	app "github.com/Zhiruosama/ai_nexus/internal"
	_ "github.com/Zhiruosama/ai_nexus/internal/grpc"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
)

func main() {
	app.Run()
}
