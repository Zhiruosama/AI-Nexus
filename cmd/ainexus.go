// Package main 应用的入口点，初始化 Config 和 DB 配置
package main

import (
	"context"
	"log"
	"time"

	_ "github.com/Zhiruosama/ai_nexus/configs"
	app "github.com/Zhiruosama/ai_nexus/internal"
	_ "github.com/Zhiruosama/ai_nexus/internal/grpc"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	rabbitmq "github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	websocket "github.com/Zhiruosama/ai_nexus/internal/pkg/ws"
)

func main() {
	rabbitmq.GlobalMQ.Close()
	websocket.GlobalHub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rabbitmq.GlobalMQ.WaitForConnection(ctx); err != nil {
		log.Fatalf("[Main] Failed to wait for RabbitMQ connection: %v\n", err)
	}

	app.StartWorker(3, app.StartText2ImgWorker)
	app.StartWorker(2, app.StartImg2ImgWorker)
	app.StartWorker(2, app.StartDeadLetterWorker)

	app.Run()
}
