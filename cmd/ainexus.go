// Package main 应用的入口点，初始化 Config 和 DB 配置
package main

import (
	_ "github.com/Zhiruosama/ai_nexus/configs"
	app "github.com/Zhiruosama/ai_nexus/internal"
	_ "github.com/Zhiruosama/ai_nexus/internal/grpc"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	rabbitmq "github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	websocket "github.com/Zhiruosama/ai_nexus/internal/pkg/ws"
)

func main() {
	defer rabbitmq.GlobalMQ.Close()
	defer websocket.GlobalHub.Close()

	for !rabbitmq.GlobalMQ.IsConnected() {
	}

	app.StartWorker(3, app.StartText2ImgWorker)
	app.StartWorker(2, app.StartImg2ImgWorker)

	app.Run()
}
