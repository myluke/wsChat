package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conversations      = make(map[string][]*websocket.Conn)
	conversationsMutex sync.Mutex
	nextConversationId = 1
)

type Message struct {
	Type           string    `json:"type"`
	Content        string    `json:"content"`
	ConversationId string    `json:"conversationId,omitempty"`
	SentTime       time.Time `json:"sentTime"`
	ReceivedTime   time.Time `json:"receivedTime"`
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		var msg Message
		err = json.Unmarshal(p, &msg)
		if err != nil {
			log.Println(err)
			continue
		}

		receivedTime := time.Now()

		switch msg.Type {
		case "newConversation":
			conversationsMutex.Lock()
			conversationId := fmt.Sprintf("%d", nextConversationId)
			nextConversationId++
			conversations[conversationId] = []*websocket.Conn{conn}
			conversationsMutex.Unlock()
			response := Message{
				Type:         "newConversation",
				Content:      conversationId,
				SentTime:     time.Now(),
				ReceivedTime: receivedTime,
			}
			err = conn.WriteJSON(response)
		case "chat":
			conversationsMutex.Lock()
			if conns, ok := conversations[msg.ConversationId]; ok {
				// 创建服务器回复消息
				serverResponse := Message{
					Type:           "chat",
					Content:        fmt.Sprintf("服务端：%s", msg.Content),
					ConversationId: msg.ConversationId,
					SentTime:       time.Now(),
					ReceivedTime:   receivedTime,
				}
				// 更新原始消息的接收时间
				msg.ReceivedTime = receivedTime
				// 发送原始消息和服务器回复给所有连接
				for _, c := range conns {
					err = c.WriteJSON(msg)
					if err != nil {
						log.Println(err)
					}
					err = c.WriteJSON(serverResponse)
					if err != nil {
						log.Println(err)
					}
				}
			}
			conversationsMutex.Unlock()
		case "loadHistory":
			// 这里可以添加加载历史记录的逻辑
			// 现在我们只发送一条欢迎消息
			response := Message{
				Type:         "chat",
				Content:      "欢迎来到会话 " + msg.ConversationId,
				SentTime:     time.Now(),
				ReceivedTime: receivedTime,
			}
			err = conn.WriteJSON(response)
		}

		if err != nil {
			log.Println(err)
			return
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("Server is running on :8880")
	log.Fatal(http.ListenAndServe(":8880", nil))
}
