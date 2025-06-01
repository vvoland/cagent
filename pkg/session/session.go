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
	Agent   *agent.Agent               `json:"agent"`
	Message chat.ChatCompletionMessage `json:"message"`
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

func (s *Session) GetMessages(a *agent.Agent) []chat.ChatCompletionMessage {
	agentSession, exists := s.Agents[a.Name()]
	if !exists {
		return nil
	}

	messages := make([]chat.ChatCompletionMessage, 0)
	contextMessages := make([]chat.ChatCompletionMessage, 0)

	if agentSession.HasSubAgents() || agentSession.HasParents() {
		subAgents := append(agentSession.SubAgents(), agentSession.Parents()...)

		subAgentsStr := ""
		for _, subAgent := range subAgents {
			subAgentsStr += subAgent.Name() + ": " + subAgent.Description() + "\n"
		}

		// messages = append(messages, chat.ChatCompletionMessage{
		// 	Role:    "system",
		// 	Content: "You are a multi-agent system, make sure to answer the user query in the most helpful way possible. You have access to these sub-agents: " + subAgentsStr + "\n\nIf you are the best to answer the question according to your description, you\ncan answer it.\n\nIf another agent is better for answering the question according to its\ndescription, call `transfer_to_agent` function to transfer the\nquestion to that agent. When transferring, do not generate any text other than\nthe function call.\n\n",
		// })
	}

	date := ""
	if a.AddDate() {
		date = "Date today is: " + time.Now().Format("2006-01-02") + "\n"
	}

	messages = append(messages, chat.ChatCompletionMessage{
		Role:    "system",
		Content: agentSession.Instruction() + "\n\n" + date,
	})

	for _, msg := range s.Messages {
		if msg.Message.Role == "system" {
			continue
		}

		if msg.Message.Role == "assistant" && msg.Agent != a {
			messages = append(messages, msg.Message)

			if len(msg.Message.ToolCalls) == 0 {
				content := fmt.Sprintf("[%s] said: %s", msg.Agent.Name(), msg.Message.Content)

				contextMessages = append(contextMessages, chat.ChatCompletionMessage{
					Role: "user",
					MultiContent: []chat.ChatMessagePart{
						{
							Type: chat.ChatMessagePartTypeText,
							Text: "For context:",
						},
						{
							Type: chat.ChatMessagePartTypeText,
							Text: content,
						},
					},
				})
			}
			continue
		}

		if msg.Message.Role == "tool" {
			messages = append(messages, msg.Message)
			content := fmt.Sprintf("For context: [%s] Tool %s returned: %s", msg.Agent.Name(), msg.Message.ToolCallID, msg.Message.Content)
			contextMessages = append(contextMessages, chat.ChatCompletionMessage{
				Role:    "user",
				Content: content,
			})
			continue
		}

		messages = append(messages, msg.Message)
	}

	messages = append(messages, contextMessages...)

	return messages
}
