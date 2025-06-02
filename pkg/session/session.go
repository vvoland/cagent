package session

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/chat"
)

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
}

// AgentMessage is a message from an agent
type AgentMessage struct {
	Agent   *agent.Agent `json:"agent"`
	Message chat.Message `json:"message"`
}

// New creates a new agent session
func New(agents map[string]*agent.Agent) *Session {
	return &Session{
		ID:        uuid.New().String(),
		State:     make(map[string]any),
		Agents:    agents,
		CreatedAt: time.Now(),
	}
}

func (s *Session) GetMessages(a *agent.Agent) []chat.Message {
	agentSession, exists := s.Agents[a.Name()]
	if !exists {
		return nil
	}

	messages := make([]chat.Message, 0)
	contextMessages := make([]chat.Message, 0)

	if agentSession.HasSubAgents() || agentSession.HasParents() {
		subAgents := append(agentSession.SubAgents(), agentSession.Parents()...)

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
		Content: agentSession.Instruction() + "\n\n" + date,
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

	// messages = append(messages, contextMessages...)

	return messages
}
