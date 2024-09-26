package main

import (
	"log"
	"net/http"
	"wstest/handlers"
	"wstest/websocket"
)

func main() {
	// 步骤1: 创建一个新的 WebSocket 管理器
	// 好处: 集中管理所有的 WebSocket 连接和会话
	manager := websocket.NewManager()

	// 步骤2: 在后台运行管理器
	// 好处: 允许管理器在独立的 goroutine 中处理消息，不阻塞主线程
	go manager.Run()

	// 步骤3: 设置 WebSocket 处理路由
	// 好处: 将 WebSocket 连接的处理委托给专门的处理函数
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeWs(manager, w, r)
	})

	// 步骤4: 启动 HTTP 服务器
	// 好处: 在指定端口上监听 WebSocket 连接请求
	log.Println("Server is running on :8880")
	log.Fatal(http.ListenAndServe(":8880", nil))
}
