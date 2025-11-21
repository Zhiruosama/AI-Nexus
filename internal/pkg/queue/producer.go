package queue

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publish 发送任务消息到队列
func Publish(ctx context.Context, taskType int, message *TaskMessage) error {
	// 获取通道
	ch, err := GlobalMQ.GetChannel()
	if err != nil {
		return err
	}

	// 启用发布确认模式
	if err = ch.Confirm(false); err != nil {
		return err
	}

	// 序列化消息
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	var routingKey string
	switch taskType {
	case 1:
		routingKey = RoutingKeyText2Img
	case 2:
		routingKey = RoutingKeyImg2Img
	default:
		return errors.New("invalid task type")
	}

	// 发送消息并获取确认
	confirmation, err := ch.PublishWithDeferredConfirmWithContext(ctx, ExchangeGenImg, routingKey, false, false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			MessageId:    message.TaskID,
			Timestamp:    time.Now(),
			ContentType:  "application/json",
			Body:         body,
		},
	)
	if err != nil {
		return err
	}

	// 等待确认
	if !confirmation.Wait() {
		return errors.New("RabbitMQ publish not confirmed")
	}

	log.Printf("[RabbitMQ] Message published successfully, task_id: %s, routing_key: %s\n", message.TaskID, routingKey)

	return nil
}
