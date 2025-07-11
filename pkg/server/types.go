package server

import "github.com/rumpl/cagent/pkg/chat"

type Message struct {
	Role    chat.MessageRole `json:"role"`
	Content string           `json:"content"`
}
