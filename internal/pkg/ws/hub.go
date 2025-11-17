package ws

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Hub 管理所有 WebSocket 连接 user_uuid -> *Client
type Hub struct {
	clients    sync.Map
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
}

// GlobalHub 全局 WebSocket Hub 实例
var GlobalHub *Hub

// 初始化全局 Hub
func init() {
	GlobalHub = &Hub{
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
		done:       make(chan struct{}),
	}
	go GlobalHub.Run()
	log.Println("[WebSocket] Hub initialized and running")
}

// Run 启动 Hub 主循环
func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			log.Println("[WebSocket] Hub shutting down")
			return
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// Register 注册新客户端到 Hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// UnRegister 注销客户端
func (h *Hub) UnRegister(client *Client) {
	h.unregister <- client
}

// SendToUser 向指定用户发送消息
func (h *Hub) SendToUser(userUUID string, msgType MessageType, data any) {
	message := Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		UserUUID:  userUUID,
	}
	h.broadcast <- message
}

// GetOnlineUserCount 获取在线用户数
func (h *Hub) GetOnlineUserCount() int {
	count := 0
	h.clients.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// IsUserOnline 检查用户是否在线
func (h *Hub) IsUserOnline(userUUID string) bool {
	_, ok := h.clients.Load(userUUID)
	return ok
}

// Close 关闭 Hub
func (h *Hub) Close() {
	close(h.done)

	h.clients.Range(func(_, value any) bool {
		client := value.(*Client)
		client.closeSend()
		return true
	})

	log.Println("[WebSocket] Hub closed")
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	if oldClient, exists := h.clients.LoadAndDelete(client.UserUUID); exists {
		old := oldClient.(*Client)
		old.closeSend()
		log.Printf("[WebSocket] Closed old connection for user: %s\n", client.UserUUID)
	}

	h.clients.Store(client.UserUUID, client)
	log.Printf("[WebSocket] User connected: %s\n", client.UserUUID)

	h.SendToUser(client.UserUUID, MessageTypeConnected, ConnectedData{
		SuccessMsg: fmt.Sprintf("Websocket connect successful: %s", client.UserUUID),
	})
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	if _, exists := h.clients.LoadAndDelete(client.UserUUID); exists {
		client.closeSend()
		log.Printf("[WebSocket] User disconnected: %s\n", client.UserUUID)
	}
}

// broadcastMessage 广播消息到指定用户
func (h *Hub) broadcastMessage(message Message) {
	if client, ok := h.clients.Load(message.UserUUID); ok {
		c := client.(*Client)

		select {
		case c.send <- message:
		default:
			log.Printf("[WebSocket] Send channel full, closing connection for user: %s\n", message.UserUUID)
			h.clients.Delete(message.UserUUID)
			c.closeSend()
		}
	}
}
