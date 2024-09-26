package handlers

import (
	"log"
	"net/http"
	"wstest/websocket"

	gws "github.com/gorilla/websocket"
)

// 步骤1: 设置 WebSocket 升级器
// 好处: 允许从 HTTP 连接升级到 WebSocket 连接
var upgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 在生产环境中应该进行更严格的检查
	},
}

// ServeWs 处理 WebSocket 连接请求
func ServeWs(manager *websocket.Manager, w http.ResponseWriter, r *http.Request) {
	// 步骤2: 将 HTTP 连接升级为 WebSocket 连接
	// 好处: 建立持久的双向通信通道
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// 从查询参数或头部获取客户端地址
	address := r.URL.Query().Get("address")
	if address == "" {
		address = r.Header.Get("X-Client-Address")
	}
	if address == "" {
		log.Println("Client address not provided")
		conn.Close()
		return
	}

	// 步骤3: 创建新的客户端
	// 好处: 为每个连接创建一个独立的客户端对象，方便管理
	client := &websocket.Client{
		Manager: manager,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Address: address,
	}

	// 步骤4: 将新客户端注册到管理器
	// 好处: 允许管理器跟踪所有活动连接
	manager.Register <- client

	// 步骤5: 启动客户端的读写 goroutines
	// 好处: 异步处理每个客户端的消息收发，提高并发性能
	go client.WritePump()
	go client.ReadPump()
}
