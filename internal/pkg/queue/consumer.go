// Package queue 包含 RabbitMQ 的消费者
package queue

import (
	"encoding/json"
	"log"
)

// MessageHandler 消息处理函数类型
type MessageHandler func(*TaskMessage) error

// Consume 从指定队列消费消息
func Consume(queueName string, handler MessageHandler) error {
	// 创建新的独立通道
	ch, err := GlobalMQ.NewChannel()
	if err != nil {
		return err
	}
	defer func() {
		if err := ch.Close(); err != nil {
			log.Printf("[RabbitMQ] Failed to close channel: %v\n", err)
		}
	}()

	// 设置 QoS
	err = ch.Qos(1, 0, false)
	if err != nil {
		return err
	}

	// 监听消息
	msgs, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	for msg := range msgs {
		var taskMsg TaskMessage

		if err := json.Unmarshal(msg.Body, &taskMsg); err != nil {
			log.Printf("[RabbitMQ] Failed to unmarshal message for task_id %s: %v\n", msg.MessageId, err)
			if err := msg.Nack(false, false); err != nil {
				log.Printf("[RabbitMQ] Failed to nack message for task_id %s: %v\n", msg.MessageId, err)
			}
			continue
		}

		err := handler(&taskMsg)
		if err != nil {
			log.Printf("[RabbitMQ] Handler failed for task_id %s: %v\n", msg.MessageId, err)
			// TODO: 此处只是直接重新入队了,后续需要区分可重试和不可重试错误
			if err := msg.Nack(false, true); err != nil {
				log.Printf("[RabbitMQ] Failed to nack message for task_id %s: %v\n", msg.MessageId, err)
			}
		} else {
			if err := msg.Ack(false); err != nil {
				log.Printf("[RabbitMQ] Failed to ack message for task_id %s: %v\n", msg.MessageId, err)
			}
		}
	}

	return nil
}
