package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Message 结构体定义
type Message struct {
	Seq  int         `json:"seq"`
	Cmd  string      `json:"cmd"`
	Body interface{} `json:"body"`
}

// MessageBody 结构体定义，用于 msg.send 命令
type MessageBody struct {
	Recipient   string `json:"recipient"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
}

// ListBody 结构体定义，用于 msg.list 命令
type ListBody struct {
	Cursor string `json:"cursor"`
}

// 新增 Conversation 结构体
type Conversation struct {
	ID          string    `json:"id"`
	Address1    string    `json:"address1"`
	Address2    string    `json:"address2"`
	ContentType string    `json:"content_type"`
	LastMsg     string    `json:"last_msg"`
	Updated     time.Time `json:"updated"`
}

// 新增 ConversationSettings 结构体
type ConversationSettings struct {
	Nickname   string    `json:"nickname"`
	Pinned     bool      `json:"pinned"`
	Blocked    bool      `json:"blocked"`
	LastSynced string    `json:"last_synced"`
	LastSeen   time.Time `json:"last_seen"`
}

// 新增 ConversationResponse 结构体
type ConversationResponse struct {
	Conversation Conversation          `json:"conversation"`
	Settings     *ConversationSettings `json:"settings"`
}

type Manager struct {
	Clients        map[*Client]bool
	Broadcast      chan BroadcastMessage
	Register       chan *Client
	Unregister     chan *Client
	Conversations  map[string]*Conversation
	ConversationMu sync.RWMutex
	NextConvID     int
}

// NewManager 创建一个新的 WebSocket 管理器
func NewManager() *Manager {
	return &Manager{
		Clients:       make(map[*Client]bool),
		Broadcast:     make(chan BroadcastMessage),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		Conversations: make(map[string]*Conversation),
		NextConvID:    1,
	}
}

// Run 运行 WebSocket 管理器的主循环
func (m *Manager) Run() {
	for {
		select {
		// 步骤1: 处理新客户端注册
		// 好处: 跟踪所有活动连接
		case client := <-m.Register:
			m.Clients[client] = true
		// 步骤2: 处理客户端注销
		// 好处: 及时清理断开的连接
		case client := <-m.Unregister:
			if _, ok := m.Clients[client]; ok {
				delete(m.Clients, client)
				close(client.Send)
				m.removeClientFromConversations(client)
			}
		// 步骤3: 处理广播消息
		// 好处: 集中处理所有接收到的消息
		case broadcastMsg := <-m.Broadcast:
			m.handleMessage(broadcastMsg.data, broadcastMsg.client)
		}
	}
}

// handleMessage 处理接收到的消息
func (m *Manager) handleMessage(message []byte, client *Client) {
	log.Printf("Received message: %s", string(message))
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	log.Printf("Processed message: %+v", msg)

	switch msg.Cmd {
	case "msg.send":
		var body MessageBody
		bodyBytes, err := json.Marshal(msg.Body)
		if err != nil {
			log.Printf("Error marshaling message body: %v", err)
			return
		}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			log.Printf("Error unmarshaling message body: %v", err)
			return
		}
		m.handleSendMessage(msg.Seq, body, client)
	case "msg.list":
		var body ListBody
		bodyBytes, err := json.Marshal(msg.Body)
		if err != nil {
			log.Printf("Error marshaling list body: %v", err)
			return
		}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			log.Printf("Error unmarshaling list body: %v", err)
			return
		}
		m.handleListConversations(msg.Seq, body, client)
	case "heartbeat":
		m.handleHeartbeat(client)
	default:
		log.Printf("Unknown message command: %s", msg.Cmd)
	}
}

// handleSendMessage 处理发送消息请求
func (m *Manager) handleSendMessage(seq int, body MessageBody, client *Client) {
	// 创建或更新会话
	convID := m.getOrCreateConversation(client.Address, body.Recipient)

	// 更新会话信息
	m.ConversationMu.Lock()
	conv := m.Conversations[convID]
	conv.LastMsg = body.Body
	conv.ContentType = body.ContentType
	conv.Updated = time.Now()
	m.ConversationMu.Unlock()

	// 发送消息给接收者
	m.sendMessageToRecipient(seq, body, convID)
}

// getOrCreateConversation 获取或创建会话
func (m *Manager) getOrCreateConversation(address1, address2 string) string {
	m.ConversationMu.Lock()
	defer m.ConversationMu.Unlock()

	// 查找现有会话
	for id, conv := range m.Conversations {
		if (conv.Address1 == address1 && conv.Address2 == address2) ||
			(conv.Address1 == address2 && conv.Address2 == address1) {
			return id
		}
	}

	// 创建新会话
	convID := fmt.Sprintf("%d", m.NextConvID)
	m.NextConvID++
	m.Conversations[convID] = &Conversation{
		ID:       convID,
		Address1: address1,
		Address2: address2,
	}
	return convID
}

// sendMessageToRecipient 发送消息给接收者
func (m *Manager) sendMessageToRecipient(seq int, body MessageBody, convID string) {
	m.ConversationMu.RLock()
	conv, ok := m.Conversations[convID]
	m.ConversationMu.RUnlock()

	if ok {
		for client := range m.Clients {
			if client.Address == conv.Address1 || client.Address == conv.Address2 {
				response := Message{
					Seq:  seq,
					Cmd:  "msg.send",
					Body: body,
				}
				select {
				case client.Send <- m.encodeMessage(response):
				default:
					close(client.Send)
					delete(m.Clients, client)
				}
			}
		}
	}
}

// handleListConversations 处理获取会话列表请求
func (m *Manager) handleListConversations(seq int, body ListBody, client *Client) {
	m.ConversationMu.RLock()
	defer m.ConversationMu.RUnlock()

	var conversations []ConversationResponse
	for _, conv := range m.Conversations {
		if conv.Address1 == client.Address || conv.Address2 == client.Address {
			resp := ConversationResponse{
				Conversation: *conv,
				Settings:     nil, // 在实际应用中，这里应该从数据库获取设置
			}
			conversations = append(conversations, resp)
		}
	}

	response := Message{
		Seq:  seq,
		Cmd:  "msg.list",
		Body: conversations,
	}

	client.Send <- m.encodeMessage(response)
}

// broadcastToAll 向所有客户端广播消息
func (m *Manager) broadcastToAll(msg Message) {
	log.Printf("Broadcasting to all: %+v", msg)
	for client := range m.Clients {
		select {
		case client.Send <- m.encodeMessage(msg):
			log.Printf("Sent message to client")
		default:
			log.Printf("Failed to send message to client, closing connection")
			close(client.Send)
			delete(m.Clients, client)
		}
	}
}

// broadcastToConversation 向特定会话的所有客户端广播消息
func (m *Manager) broadcastToConversation(convID string, msg Message) {
	m.ConversationMu.RLock()
	conv, ok := m.Conversations[convID]
	m.ConversationMu.RUnlock()

	if ok {
		for client := range m.Clients {
			if client.Address == conv.Address1 || client.Address == conv.Address2 {
				select {
				case client.Send <- m.encodeMessage(msg):
				default:
					close(client.Send)
					delete(m.Clients, client)
				}
			}
		}
	}
}

// removeClientFromConversations 从所有会话中移除客户端
func (m *Manager) removeClientFromConversations(client *Client) {
	m.ConversationMu.Lock()
	defer m.ConversationMu.Unlock()

	for _, conv := range m.Conversations {
		if conv.Address1 == client.Address || conv.Address2 == client.Address {
			delete(m.Conversations, conv.ID)
		}
	}
}

// 处理心跳
func (m *Manager) handleHeartbeat(client *Client) {
	if client == nil {
		log.Println("警告: 收到来自空客户端的心跳")
		return
	}

	log.Printf("收到来自客户端 %p 的心跳", client)

	// 准备心跳响应
	response := Message{
		Seq:  0,
		Cmd:  "heartbeat",
		Body: "pong",
	}

	// 直接发送响应给客户端，而不是广播
	select {
	case client.Send <- m.encodeMessage(response):
		log.Printf("已发送心跳响应给客户端 %p", client)
	default:
		log.Printf("无法发送心跳响应给客户端 %p，可能连接已关闭", client)
		delete(m.Clients, client)
		close(client.Send)
	}
}

type BroadcastMessage struct {
	data   []byte
	client *Client
}

// 新增 encodeMessage 方法
func (m *Manager) encodeMessage(msg interface{}) []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error encoding message: %v", err)
		return nil
	}
	return data
}
