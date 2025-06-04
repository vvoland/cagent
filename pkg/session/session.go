package session

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/chat"
)

var maxMessages = 20 // Maximum number of messages to keep in context

// Session represents the agent's state including conversation history and variables
type Session struct {
	// ID is the unique identifier for the session
	ID string `json:"id"`

	// Each agent in a multi-agent system has its own session
	Agents map[string]*agent.Agent `json:"agents"`

	// Messages holds the conversation history
	Messages []AgentMessage `json:"messages"`

	// State is a general-purpose map to store arbitrary state data, it is shared between agents
	State map[string]any `json:"state"`

	// CreatedAt is the time the session was created
	CreatedAt time.Time `json:"created_at"`

	// Logger for debugging and logging session operations
	logger *slog.Logger
}

// AgentMessage is a message from an agent
type AgentMessage struct {
	Agent   *agent.Agent `json:"agent"`
	Message chat.Message `json:"message"`
}

// New creates a new agent session
func New(agents map[string]*agent.Agent, logger *slog.Logger) *Session {
	sessionID := uuid.New().String()
	logger.Debug("Creating new session", "session_id", sessionID)

	return &Session{
		ID:        sessionID,
		State:     make(map[string]any),
		Agents:    agents,
		CreatedAt: time.Now(),
		logger:    logger,
	}
}

func (s *Session) GetMessages(a *agent.Agent) []chat.Message {
	s.logger.Debug("Getting messages for agent", "agent", a.Name(), "session_id", s.ID)

	messages := make([]chat.Message, 0)
	contextMessages := make([]chat.Message, 0)

	if a.HasSubAgents() || a.HasParents() {
		subAgents := append(a.SubAgents(), a.Parents()...)

		subAgentsStr := ""
		for _, subAgent := range subAgents {
			subAgentsStr += subAgent.Name() + ": " + subAgent.Description() + "\n"
		}

		messages = append(messages, chat.Message{
			Role:    "system",
			Content: "You are a multi-agent system, make sure to answer the user query in the most helpful way possible. You have access to these sub-agents: " + subAgentsStr + "\n\nIf you are the best to answer the question according to your description, you\ncan answer it.\n\nIf another agent is better for answering the question according to its\ndescription, call `transfer_to_agent` function to transfer the\nquestion to that agent. When transferring, do not generate any text other than\nthe function call.\n\n",
		})
	}

	date := ""
	if a.AddDate() {
		date = "Date today is: " + time.Now().Format("2006-01-02") + "\n"
	}

	messages = append(messages, chat.Message{
		Role:    "system",
		Content: a.Instruction() + "\n\n" + date,
	})

	for _, tool := range a.ToolImpls() {
		if tool.Instructions() != "" {
			messages = append(messages, chat.Message{
				Role:    "system",
				Content: tool.Instructions(),
			})
		}
	}

	for i := range s.Messages {
		if s.Messages[i].Message.Role == "system" {
			continue
		}

		if s.Messages[i].Message.Role == "assistant" && s.Messages[i].Agent != a {
			messages = append(messages, s.Messages[i].Message)

			if len(s.Messages[i].Message.ToolCalls) == 0 {
				content := fmt.Sprintf("[%s] said: %s", s.Messages[i].Agent.Name(), s.Messages[i].Message.Content)

				contextMessages = append(contextMessages, chat.Message{
					Role: "user",
					MultiContent: []chat.MessagePart{
						{
							Type: chat.MessagePartTypeText,
							Text: "For context:",
						},
						{
							Type: chat.MessagePartTypeText,
							Text: content,
						},
					},
				})
			}
			continue
		}

		if s.Messages[i].Message.Role == "tool" && s.Messages[i].Agent != a {
			messages = append(messages, s.Messages[i].Message)
			content := fmt.Sprintf("For context: [%s] Tool %s returned: %s", s.Messages[i].Agent.Name(), s.Messages[i].Message.ToolCallID, s.Messages[i].Message.Content)
			contextMessages = append(contextMessages, chat.Message{
				Role:    "user",
				Content: content,
			})
			continue
		}

		messages = append(messages, s.Messages[i].Message)
	}

	messages = append(messages, contextMessages...)
	trimmed := trimMessages(messages)

	s.logger.Debug("Retrieved messages for agent",
		"agent", a.Name(),
		"session_id", s.ID,
		"total_messages", len(messages),
		"trimmed_messages", len(trimmed))

	return trimmed
}

// trimMessages ensures we don't exceed the maximum number of messages while maintaining
// consistency between assistant messages and their tool call results
func trimMessages(messages []chat.Message) []chat.Message {
	if len(messages) <= maxMessages {
		return messages
	}

	// Keep track of tool call IDs that need to be removed
	toolCallsToRemove := make(map[string]bool)

	// Calculate how many messages we need to remove
	toRemove := len(messages) - maxMessages

	// Start from the beginning (oldest messages)
	for i := 0; i < toRemove; i++ {
		// If this is an assistant message with tool calls, mark them for removal
		if messages[i].Role == "assistant" {
			for _, toolCall := range messages[i].ToolCalls {
				toolCallsToRemove[toolCall.ID] = true
			}
		}
	}

	// Filter messages keeping only those we want to keep
	result := make([]chat.Message, 0, maxMessages)
	for i := toRemove; i < len(messages); i++ {
		msg := messages[i]

		// Skip tool messages that correspond to removed assistant messages
		if msg.Role == "tool" && toolCallsToRemove[msg.ToolCallID] {
			continue
		}

		result = append(result, msg)
	}

	return result
}
