package session

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/sashabaranov/go-openai"
)

// Session represents the agent's state including conversation history and variables
type Session struct {
	// ID is the unique identifier for the session
	ID string

	// Each agent in a multi-agent system has its own session
	AgentSession map[string]*AgentSession

	// Messages holds the conversation history
	Messages []AgentMessage

	// State is a general-purpose map to store arbitrary state data, it is shared between agents
	State map[string]any

	cfg *config.Config
}

// AgentMessage is a message from an agent
type AgentMessage struct {
	Agent   *agent.Agent
	Message openai.ChatCompletionMessage
}

type AgentSession struct {
	// Agent is the agent that this session belongs to
	Agent *agent.Agent
	// Messages holds the conversation history
	Messages []AgentMessage
}

// New creates a new agent session
func New(cfg *config.Config) *Session {
	return &Session{
		ID:           uuid.New().String(),
		State:        make(map[string]any),
		AgentSession: make(map[string]*AgentSession),
		cfg:          cfg,
	}
}

func (s *Session) GetMessages(a *agent.Agent) []openai.ChatCompletionMessage {
	// Get the agent session
	agentSession, exists := s.AgentSession[a.Name()]
	if !exists {
		agentSession = &AgentSession{
			Agent:    a,
			Messages: make([]AgentMessage, 0),
		}
		s.AgentSession[a.Name()] = agentSession
	}

	// Create a new slice to hold the processed messages
	messages := make([]openai.ChatCompletionMessage, 0)

	if agentSession.Agent.HasSubAgents() {
		subAgents := agentSession.Agent.SubAgents()
		subAgentsStr := ""
		for _, subAgent := range subAgents {
			subAgentSession, exists := s.AgentSession[subAgent.Name()]
			if !exists {
				aa, _ := agent.New(subAgent.Name(), s.cfg.Agents[subAgent.Name()].Instruction)
				subAgentSession = &AgentSession{
					Agent:    aa,
					Messages: make([]AgentMessage, 0),
				}
				s.AgentSession[subAgent.Name()] = subAgentSession
			}
			subAgentsStr += subAgent.Name() + ": " + subAgentSession.Agent.Description() + "\n"
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "system",
			Content: "You are a multi-agent system, make sure to answer the user query in the most helpful way possible. You have access to these sub-agents: " + subAgentsStr + "\n\nIf you are the best to answer the question according to your description, you\ncan answer it.\n\nIf another agent is better for answering the question according to its\ndescription, call `transfer_to_agent` function to transfer the\nquestion to that agent. When transferring, do not generate any text other than\nthe function call.\n",
		})
	}

	// Add the agent's system prompt as the first message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    "system",
		Content: agentSession.Agent.Instruction(),
	})

	for _, msg := range s.Messages {
		if msg.Message.Role == "system" {
			continue
		}

		if msg.Message.Role == "assistant" {
			messages = append(messages, msg.Message)

			if len(msg.Message.ToolCalls) == 0 {
				content := fmt.Sprintf("[%s] said: %s", msg.Agent.Name(), msg.Message.Content)

				messages = append(messages, openai.ChatCompletionMessage{
					Role: "user",
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: "For context:",
						},
						{
							Type: openai.ChatMessagePartTypeText,
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
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    "user",
				Content: content,
			})
			continue
		}

		messages = append(messages, msg.Message)
	}

	return messages
}
