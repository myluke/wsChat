package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Manager *Manager
	Conn    *websocket.Conn
	Send    chan []byte
	Address string // 添加这个字段
}

const (
	// 写入超时
	writeWait = 10 * time.Second

	// 允许读取下一个pong消息的时间
	pongWait = 60 * time.Second

	// 发送ping的频率应小于pongWait
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512
)

// ReadPump 持续从 WebSocket 连接读取消息
func (c *Client) ReadPump() {
	defer func() {
		// 步骤1: 清理资源
		// 好处: 确保在连接关闭时正确清理资源
		if c.Manager != nil {
			c.Manager.Unregister <- c
		}
		c.Conn.Close()
	}()

	// 设置读取限制和超时
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// 步骤2: 读取消息
		// 好处: 持续监听客户端发送的消息
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
			// 步骤3: 将消息广播到管理器
			// 好处: 集中处理所有接收到的消息
		}
		log.Printf("Received message from client: %s", string(message))

		if c.Manager != nil {
			c.Manager.Broadcast <- BroadcastMessage{data: message, client: c}
		} else {
			log.Println("Manager is nil, cannot broadcast message")
		}
	}
}

// WritePump 持续向 WebSocket 连接写入消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			// 步骤1: 检查通道是否已关闭
			// 好处: 优雅地处理通道关闭的情况
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 步骤2: 写入消息
			// 好处: 将消发送给客户端
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			log.Printf("Sent message to client: %s", string(message))

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			// 发送 ping 消息
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
