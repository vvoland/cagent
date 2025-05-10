package session

import (
	"github.com/sashabaranov/go-openai"
)

// Session represents the agent's state including conversation history and variables
type Session struct {
	// Messages holds the conversation history
	Messages []openai.ChatCompletionMessage

	// State is a general-purpose map to store arbitrary state data
	State map[string]any
}

// New creates a new agent session
func New() *Session {
	return &Session{
		Messages: []openai.ChatCompletionMessage{},
		State:    make(map[string]any),
	}
}

// AddMessage adds a message to the conversation history
func (s *Session) AddMessage(message openai.ChatCompletionMessage) {
	s.Messages = append(s.Messages, message)
}

// GetMessages returns the conversation history
func (s *Session) GetMessages() []openai.ChatCompletionMessage {
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
