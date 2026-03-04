// Package main 应用的入口点，初始化 Config 和 DB 配置
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/Zhiruosama/ai_nexus/configs"
	app "github.com/Zhiruosama/ai_nexus/internal"
	_ "github.com/Zhiruosama/ai_nexus/internal/grpc"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	rabbitmq "github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	websocket "github.com/Zhiruosama/ai_nexus/internal/pkg/ws"
)

func main() {
	// 命令行参数解析
	port := flag.Int("p", 0, "服务端口号，不指定则使用配置文件中的端口")
	flag.Parse()

	// 如果指定了端口，覆盖配置文件中的端口
	if *port > 0 {
		configs.GlobalConfig.Server.Port = *port
		log.Printf("[Main] 使用命令行指定端口: %d\n", *port)
	}
	defer rabbitmq.GlobalMQ.Close()
	defer websocket.GlobalHub.Close()

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
