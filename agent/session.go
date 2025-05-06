package agent

import (
	goOpenAI "github.com/sashabaranov/go-openai"
)

// Session represents the agent's state including conversation history and variables
type Session struct {
	// Messages holds the conversation history
	Messages []goOpenAI.ChatCompletionMessage

	// State is a general-purpose map to store arbitrary state data
	State map[string]any
}

// NewSession creates a new agent session
func NewSession() *Session {
	return &Session{
		Messages: []goOpenAI.ChatCompletionMessage{},
		State:    make(map[string]any),
	}
}

// AddMessage adds a message to the conversation history
func (s *Session) AddMessage(message goOpenAI.ChatCompletionMessage) {
	s.Messages = append(s.Messages, message)
}

// GetMessages returns the conversation history
func (s *Session) GetMessages() []goOpenAI.ChatCompletionMessage {
	return s.Messages
}

// SetState sets a value in the state map
func (s *Session) SetState(key string, value any) {
	s.State[key] = value
}

// GetState retrieves a value from the state map
func (s *Session) GetState(key string) (any, bool) {
	value, exists := s.State[key]
	return value, exists
}
