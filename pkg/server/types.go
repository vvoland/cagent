package server

import "github.com/docker/cagent/pkg/chat"

type Message struct {
	Role    chat.MessageRole `json:"role"`
	Content string           `json:"content"`
}
