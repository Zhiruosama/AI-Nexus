// Package queue 包含 RabbitMQ 的消费者
package queue

import (
	"encoding/json"
	"log"
	"reflect"
	"time"

	image_generation_dao "github.com/Zhiruosama/ai_nexus/internal/dao/image-generation"
)

// MessageHandler 消息处理函数类型
type MessageHandler func(*TaskMessage) (bool, int8, int8, error)
type DeadLetterHandler func(*TaskMessage, map[string]any) error

var dao = image_generation_dao.DAO{}

// Consume 从指定队列消费消息
func Consume(queueName string, handler MessageHandler) error {
	// 创建新的独立通道
	ch, err := GlobalMQ.NewChannel()
	if err != nil {
		return err
	}
	defer func() {
		if err = ch.Close(); err != nil {
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

		retry, retryCount, maxRetries, err := handler(&taskMsg)

		if err != nil {
			log.Printf("[RabbitMQ] Handler failed for task_id %s: %v\n", msg.MessageId, err)

			// 可以重试
			if retry && retryCount < maxRetries {
				// 重试
				if errs := msg.Nack(false, true); errs != nil {
					log.Printf("[RabbitMQ] Failed to nack message for task_id %s: %v\n", msg.MessageId, err)
				}

				// 更新重试次数
				if errs := dao.UpdateTaskParams("retry_count", retryCount+1, msg.MessageId); errs != nil {
					log.Printf("[RabbitMQ] Failed to update retry count for task_id %s: %v\n", msg.MessageId, err)
				}

				continue
			}

			// 不可重试
			if errs := msg.Nack(false, false); errs != nil {
				log.Printf("[RabbitMQ] Failed to nack message for task_id %s: %v\n", msg.MessageId, err)
			}

			if errs := dao.UpdateTaskParams("status", 4, msg.MessageId); errs != nil {
				log.Printf("[RabbitMQ] Failed to update status for task_id %s: %v\n", msg.MessageId, err)
			}

			if errs := dao.UpdateTaskParams("error_message", err.Error(), msg.MessageId); errs != nil {
				log.Printf("[RabbitMQ] Failed to update error message for task_id %s: %v\n", msg.MessageId, err)
			}

			if errs := dao.UpdateTaskParams("completed_at", time.Now(), msg.MessageId); errs != nil {
				log.Printf("[RabbitMQ] Failed to update error time for task_id %s: %v\n", msg.MessageId, err)
			}

		} else {
			// 消费成功
			if err := msg.Ack(false); err != nil {
				log.Printf("[RabbitMQ] Failed to ack message for task_id %s: %v\n", msg.MessageId, err)
			}
		}

	}

	return nil
}

// ConsumeDeadLetter 消费死信消息
func ConsumeDeadLetter(queueName string, handler DeadLetterHandler) error {
	// 创建新的独立通道
	ch, err := GlobalMQ.NewChannel()
	if err != nil {
		return err
	}
	defer func() {
		if err = ch.Close(); err != nil {
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
			log.Printf("[RabbitMQ] Failed to unmarshal dead letter message for task_id %s: %v\n", msg.MessageId, err)
			if err := msg.Nack(false, false); err != nil {
				log.Printf("[RabbitMQ] Failed to nack dead letter message for task_id %s: %v\n", msg.MessageId, err)
			}
			continue
		}

		xDeathInfo := extractXDeathInfo(msg.Headers)
		err := handler(&taskMsg, xDeathInfo)

		if err != nil {
			log.Printf("[RabbitMQ] Dead letter handler failed for task_id %s: %v\n", msg.MessageId, err)
			if errs := msg.Nack(false, false); errs != nil {
				log.Printf("[RabbitMQ] Failed to nack dead letter message for task_id %s: %v\n", msg.MessageId, errs)
			}
		} else {
			if err := msg.Ack(false); err != nil {
				log.Printf("[RabbitMQ] Failed to ack dead letter message for task_id %s: %v\n", msg.MessageId, err)
			}
		}
	}

	return nil
}

// extractXDeathInfo 提取 x-death 头信息
func extractXDeathInfo(headers map[string]any) map[string]any {
	result := make(map[string]any)

	if headers == nil {
		return result
	}

	xDeath, ok := headers["x-death"]
	if !ok {
		return result
	}

	xDeathSlice, ok := xDeath.([]any)
	if !ok {
		return result
	}

	if len(xDeathSlice) == 0 {
		return result
	}

	firstElement := xDeathSlice[0]
	v := reflect.ValueOf(firstElement)

	if v.Kind() != reflect.Map {
		log.Printf("[RabbitMQ] x-death element is not a map, type: %T\n", firstElement)
		return result
	}

	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key().Interface()
		value := iter.Value().Interface()

		if keyStr, ok := key.(string); ok {
			result[keyStr] = value
		}
	}

	return result
}
