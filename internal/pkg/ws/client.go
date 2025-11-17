// Package ws WebSocket 客户端
package ws

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	sendBufferSize = 256
)

// Client WebSocket 客户端
type Client struct {
	UserUUID      string
	conn          *websocket.Conn
	send          chan Message
	hub           *Hub
	closeOnce     sync.Once
	closeSendOnce sync.Once
}

// NewClient 创建新的 WebSocket 客户端
func NewClient(userUUID string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		UserUUID: userUUID,
		conn:     conn,
		send:     make(chan Message, sendBufferSize),
		hub:      hub,
	}
}

// ReadPump 读取客户端消息
func (c *Client) ReadPump() {
	defer c.close()

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("[WebSocket] Set read deadline error for user %s: %v\n", c.UserUUID, err)
		return
	}

	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("[WebSocket] Set read deadline error for user %s: %v\n", c.UserUUID, err)
		}
		return nil
	})

	for {
		// 读取消息，不过不处理，保持心跳即可
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Read error for user %s: %v\n", c.UserUUID, err)
			}
			break
		}
	}
}

// WritePump 向客户端写入消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("[WebSocket] Set write deadline error for user %s: %v\n", c.UserUUID, err)
				return
			}

			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("[WebSocket] Write close message error for user %s: %v\n", c.UserUUID, err)
				}
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("[WebSocket] Write error for user %s: %v\n", c.UserUUID, err)
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("[WebSocket] Set write deadline error for user %s: %v\n", c.UserUUID, err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[WebSocket] Ping error for user %s: %v\n", c.UserUUID, err)
				return
			}
		}
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		if err := c.conn.Close(); err != nil {
			log.Printf("[WebSocket] Close connection error for user %s: %v\n", c.UserUUID, err)
		}
		c.hub.unregister <- c
	})
}

// closeSend 安全关闭 send channel
func (c *Client) closeSend() {
	c.closeSendOnce.Do(func() {
		close(c.send)
	})
}
