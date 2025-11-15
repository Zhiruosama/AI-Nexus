// Package queue 包含 RabbitMQ 的初始化
package queue

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// ExchangeGenImg 名称
	ExchangeGenImg = "exchange.generation.image"
	// ExchangeDLX 名称
	ExchangeDLX = "exchange.generation.dlx"

	// QueueText2Img 名称
	QueueText2Img = "queue.text2img"
	// QueueImg2Img 名称
	QueueImg2Img = "queue.img2img"
	// QueueDeadLetter 名称
	QueueDeadLetter = "queue.dead_letter"

	// RoutingKeyText2Img 名称
	RoutingKeyText2Img = "generation.text2img"
	// RoutingKeyImg2Img 名称
	RoutingKeyImg2Img = "generation.img2img"
	// RoutingKeyDeadLetter 名称
	RoutingKeyDeadLetter = "dead_letter"
)

// InitQueues 初始化 RabbitMQ 队列、Exchange 和绑定关系
func InitQueues() error {
	ch, err := GlobalMQ.GetChannel()
	if err != nil {
		return err
	}

	// Exchange 声明
	// 生图 Exchange
	err = ch.ExchangeDeclare(ExchangeGenImg, "topic", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 死信 Direct Exchange
	err = ch.ExchangeDeclare(ExchangeDLX, "direct", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// Queue 声明
	// 队列参数
	queueArgs := amqp.Table{
		"x-message-ttl":             int32(1800000),       // 30分钟
		"x-max-length":              int32(1000),          // 最大队列长度
		"x-dead-letter-exchange":    ExchangeDLX,          // 死信交换机
		"x-dead-letter-routing-key": RoutingKeyDeadLetter, // 死信路由键
	}

	// 文生图队列
	_, err = ch.QueueDeclare(QueueText2Img, true, false, false, false, queueArgs)
	if err != nil {
		return err
	}

	// 图生图队列
	_, err = ch.QueueDeclare(QueueImg2Img, true, false, false, false, queueArgs)
	if err != nil {
		return err
	}

	// 死信队列, TTL: 7 天
	_, err = ch.QueueDeclare(QueueDeadLetter, true, false, false, false,
		amqp.Table{
			"x-message-ttl": int32(604800000),
		},
	)
	if err != nil {
		return err
	}

	// 绑定队列到 Exchange
	// 绑定文生图队列
	err = ch.QueueBind(QueueText2Img, RoutingKeyText2Img, ExchangeGenImg, false, nil)
	if err != nil {
		return err
	}

	// 绑定图生图队列
	err = ch.QueueBind(QueueImg2Img, RoutingKeyImg2Img, ExchangeGenImg, false, nil)
	if err != nil {
		return err
	}

	// 绑定死信队列
	err = ch.QueueBind(QueueDeadLetter, RoutingKeyDeadLetter, ExchangeDLX, false, nil)
	if err != nil {
		return err
	}

	log.Println("[RabbitMQ] All queues initialized successfully")
	return nil
}
