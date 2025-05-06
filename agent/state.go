package agent

import (
	goOpenAI "github.com/sashabaranov/go-openai"
)

// State represents the agent's state including conversation history and variables
type State struct {
	// Messages holds the conversation history
	Messages []goOpenAI.ChatCompletionMessage

	// State is a general-purpose map to store arbitrary state data
	State map[string]interface{}
}

// NewState creates a new agent state with initialized fields
func NewState() *State {
	return &State{
		Messages: []goOpenAI.ChatCompletionMessage{},
		State:    make(map[string]interface{}),
	}
}

// AddMessage adds a message to the conversation history
func (s *State) AddMessage(message goOpenAI.ChatCompletionMessage) {
	s.Messages = append(s.Messages, message)
}

// GetMessages returns the conversation history
func (s *State) GetMessages() []goOpenAI.ChatCompletionMessage {
	return s.Messages
}

// SetState sets a value in the state map
func (s *State) SetState(key string, value interface{}) {
	s.State[key] = value
}

// GetState retrieves a value from the state map
func (s *State) GetState(key string) (interface{}, bool) {
	value, exists := s.State[key]
	return value, exists
}
