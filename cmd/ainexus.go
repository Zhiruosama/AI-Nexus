// Package main 应用的入口点，初始化 Config 和 DB 配置
package main

import (
	"log"

	_ "github.com/Zhiruosama/ai_nexus/configs"
	app "github.com/Zhiruosama/ai_nexus/internal"
	_ "github.com/Zhiruosama/ai_nexus/internal/grpc"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	rabbitmq "github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
	_ "github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	websocket "github.com/Zhiruosama/ai_nexus/internal/pkg/ws"
	"github.com/rabbitmq/amqp091-go"
)

func test() {
	conn, err := amqp091.Dial("amqp://murane:845924@localhost:5672/")
	if err != nil {
		panic(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}

	defer ch.Close()

	queueName := "queue.dead_letter"

	msgs, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		panic(err)
	}

	for msg := range msgs {
		// 处理死信消息
		// 例如，记录到日志或发送到告警系统
		log.Printf("死信消息: %s", msg.Body)
		log.Printf("原因: %s", msg.Headers["x-death"])
	}
}

func main() {
	defer rabbitmq.GlobalMQ.Close()
	defer websocket.GlobalHub.Close()

	for !rabbitmq.GlobalMQ.IsConnected() {
	}

	app.StartWorker(3, app.StartText2ImgWorker)
	app.StartWorker(2, app.StartImg2ImgWorker)

	app.Run()
}
