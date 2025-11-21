package queue

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Zhiruosama/ai_nexus/configs"
	amqp "github.com/rabbitmq/amqp091-go"
)

// GlobalMQ 全局 RabbitMQ 客户端实例
var GlobalMQ *RabbitMQClient

// RabbitMQClient 是一个带自动重连的 RabbitMQ 客户端
type RabbitMQClient struct {
	url             string           // RabbitMQ 连接地址
	conn            *amqp.Connection // 连接对象
	channel         *amqp.Channel    // 通道对象
	isConnected     bool             // 连接状态
	queuesInited    bool             // 队列是否已初始化
	mu              sync.RWMutex     // 读写锁，保护连接和通道
	notifyClose     chan *amqp.Error // 监听连接关闭事件
	notifyChanClose chan *amqp.Error // 监听通道关闭事件
	done            chan bool        // 关闭信号
	ready           chan struct{}    // 连接就绪信号
	readyOnce       sync.Once        // 确保 ready 只被关闭一次
}

// GetChannel 获取通道
func (c *RabbitMQClient) GetChannel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return nil, amqp.ErrClosed
	}

	return c.channel, nil
}

// NewChannel 创建一个新的独立通道
func (c *RabbitMQClient) NewChannel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected || c.conn == nil {
		return nil, amqp.ErrClosed
	}

	return c.conn.Channel()
}

// IsConnected 检查连接状态
func (c *RabbitMQClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// WaitForConnection 等待连接就绪，支持超时控制
func (c *RabbitMQClient) WaitForConnection(ctx context.Context) error {
	if c.IsConnected() {
		return nil
	}

	select {
	case <-c.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close 关闭客户端
func (c *RabbitMQClient) Close() {
	close(c.done)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		err := c.channel.Close()
		if err != nil {
			log.Printf("[RabbitMQ] Close channel failed: %v\n", err)
		}
	}

	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			log.Printf("[RabbitMQ] Close connection failed: %v\n", err)
		}
	}

	c.isConnected = false
	log.Println("[RabbitMQ] RabbitMQ has been closed")
}

func init() {
	url := configs.GlobalConfig.RabbitMQ.URLString()
	GlobalMQ = newRabbitMQClient(url)
}

// newRabbitMQClient 创建一个新的 RabbitMQ 客户端（内部使用）
func newRabbitMQClient(url string) *RabbitMQClient {
	client := &RabbitMQClient{
		url:   url,
		done:  make(chan bool),
		ready: make(chan struct{}),
	}

	go client.handleReconnect()

	return client
}

// handleReconnect 处理连接和自动重连
func (c *RabbitMQClient) handleReconnect() {
	for {
		// 尝试建立连接
		if err := c.connect(); err != nil {
			log.Printf("[RabbitMQ] Connect to RabbitMQ failed: %v, Wait 5 seconds to retry\n", err)

			select {
			case <-c.done:
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		log.Println("[RabbitMQ] RabbitMQ connected successfully")

		if !c.queuesInited {
			if err := InitQueues(); err != nil {
				log.Printf("[RabbitMQ] Failed to initialize queues: %v\n", err)
				time.Sleep(2 * time.Second)
				continue
			}
			c.queuesInited = true
		}

		select {
		case <-c.done:
			return
		case err := <-c.notifyClose:
			c.mu.Lock()
			c.isConnected = false
			c.mu.Unlock()
			log.Printf("[RabbitMQ] Connect to RabbitMQ closed: %v\n", err)
		case err := <-c.notifyChanClose:
			log.Printf("[RabbitMQ] Channel to RabbitMQ closed: %v\n", err)
		}
	}
}

// connect 建立连接和通道
func (c *RabbitMQClient) connect() error {
	if c.isConnected {
		return nil
	}

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		if err = conn.Close(); err != nil {
			log.Printf("[RabbitMQ] Failed to close connection: %v\n", err)
		}
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.channel = ch
	c.isConnected = true
	c.mu.Unlock()

	c.readyOnce.Do(func() {
		close(c.ready)
	})

	c.notifyClose = make(chan *amqp.Error)
	c.conn.NotifyClose(c.notifyClose)

	c.notifyChanClose = make(chan *amqp.Error)
	c.channel.NotifyClose(c.notifyChanClose)

	return nil
}
