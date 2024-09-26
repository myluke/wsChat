package models

import (
	"encoding/json"
	"time"
)

type Message struct {
	Type           string    `json:"type"`
	Content        string    `json:"content"`
	ConversationId string    `json:"conversationId,omitempty"`
	SentTime       time.Time `json:"sentTime"`
	ReceivedTime   time.Time `json:"receivedTime"`
}

func (m Message) ToJSON() []byte {
	data, _ := json.Marshal(m)
	return data
}
