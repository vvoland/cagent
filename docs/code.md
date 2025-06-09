# How to Create and Run Agent Teams in Code

This guide provides practical, task-oriented instructions for creating and
running teams of AI agents using the cagent library in Go. Each section focuses
on solving specific problems with working code examples.

## Problem: Setting Up a Basic Agent

You need to create a single agent that can respond to user messages.

### Solution

```go
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/model/provider/openai"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/rumpl/cagent/pkg/team"
)

func main() {
	// Setup context and logger
	ctx := context.Background()
	logger := slog.Default()

	// Create a language model client
	llm, err := openai.NewClient(&config.ModelConfig{
		Type:  "openai",
		Model: "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a team with a single agent
	agents := team.New(map[string]*agent.Agent{
		"root": agent.New("root", "You are a helpful assistant.", agent.WithModel(llm)),
	})

	// Create a session with an initial user message
	sess := session.New(logger)
	sess.Messages = append(sess.Messages, session.UserMessage("How are you doing?"))

	// Create the runtime with our agent team
	rt, err := runtime.New(logger, agents, "root")
	if err != nil {
		log.Fatal(err)
	}

	// Run the agent and get responses
	messages, err := rt.Run(ctx, sess)
	if err != nil {
		log.Fatal(err)
	}

	// Print the final response
	fmt.Println(messages[len(messages)-1].Message.Content)
}
```

### Details

1. Import the necessary packages from the cagent library
2. Create an OpenAI GPT model client with your desired model
3. Create a single agent with a name and system instructions
4. Initialize a team with this agent
5. Create a session with a user message
6. Run the agent and print the response

This code creates the simplest possible agent setup. The agent will respond to
the user message based on its system instructions.

## Problem: Creating Multiple Independent Agents

You need to create multiple agents with different specialties.

### Solution

```go
// Create specialized agents
techAgent := agent.New(
	"tech",
	"You are a technical expert specializing in programming and technology.",
	agent.WithModel(llm)
)

creativeAgent := agent.New(
	"creative",
	"You are a creative writer specializing in storytelling and content creation.",
	agent.WithModel(llm)
)

// Create a team with both agents
agents := team.New(map[string]*agent.Agent{
	"tech": techAgent,
	"creative": creativeAgent,
})

// Create the runtime with the tech agent as the starting point
rt, err := runtime.New(logger, agents, "tech")
```

### Details

1. Create each agent with a unique name and specialized instructions
2. Add all agents to the team map
3. When creating the runtime, specify which agent to start with
4. The starting agent will handle the initial user message

This approach creates multiple independent agents. However, they cannot
communicate with each other - we'll solve that next.

## Problem: Setting Up Agent Hierarchies for Delegation

You need agents that can delegate tasks to other specialized agents.

### Solution

```go
// Create a child agent
childAgent := agent.New(
	"child",
	"You are a specialized agent with deep knowledge of historical facts.",
	agent.WithModel(llm)
)

// Create a parent agent that can delegate to the child
parentAgent := agent.New(
	"parent",
	"You are a coordinator. Delegate history questions to your child agent.",
	agent.WithModel(llm),
	agent.WithSubAgents([]*agent.Agent{childAgent})
)

// Create the team with both agents
agents := team.New(map[string]*agent.Agent{
	"parent": parentAgent,
	"child": childAgent,
})

// Start with the parent agent
rt, err := runtime.New(logger, agents, "parent")
```

### Complete Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/model/provider/openai"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/rumpl/cagent/pkg/team"
)

func main() {
	ctx := context.Background()
	logger := slog.Default()

	// Create a language model client
	llm, err := openai.NewClient(&config.ModelConfig{
		Type:  "openai",
		Model: "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a child agent
	child := agent.New("child", "You are a child agent with specialized historical knowledge.", agent.WithModel(llm))

	// Create a parent agent that can delegate to the child
	parent := agent.New(
		"parent",
		"You are a parent agent. Delegate history questions to your child agent.",
		agent.WithModel(llm),
		agent.WithSubAgents([]*agent.Agent{child})
	)

	// Create the team with both agents
	agents := team.New(map[string]*agent.Agent{
		"parent": parent,
		"child": child,
	})

	// Create a session with an initial user message
	sess := session.New(logger)
	sess.Messages = append(sess.Messages, session.UserMessage("When was the Roman Empire founded?"))

	// Create the runtime with the parent agent
	rt, err := runtime.New(logger, agents, "parent")
	if err != nil {
		log.Fatal(err)
	}

	// Run the agent team
	messages, err := rt.Run(ctx, sess)
	if err != nil {
		log.Fatal(err)
	}

	// Print the final response
	fmt.Println(messages[len(messages)-1].Message.Content)
}
```

### Details

1. Create a child agent with specialized knowledge
2. Create a parent agent with `agent.WithSubAgents([]*agent.Agent{childAgent})`
3. Add both agents to the team map
4. The parent agent can now transfer control to the child agent using the
   built-in transfer mechanism
5. The child agent will process messages until it transfers control back to the
   parent

When the parent agent receives a message about history, it can transfer control
to the child agent. The child agent will respond and then transfer control back
to the parent.

## Problem: Adding Built-in Tools to Agents

You need to extend your agent with specialized capabilities like running bash
commands.

### Solution

```go
// Create an agent with the bash tool
agent := agent.New(
	"root",
	"You are a technical expert with bash capabilities.",
	agent.WithModel(llm),
	agent.WithToolSets([]tools.ToolSet{tools.NewBashTool()})
)
```

### Complete Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/model/provider/openai"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/rumpl/cagent/pkg/team"
	"github.com/rumpl/cagent/pkg/tools"
)

func main() {
	ctx := context.Background()
	logger := slog.Default()

	// Create a language model client
	llm, err := openai.NewClient(&config.ModelConfig{
		Type:  "openai",
		Model: "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a team with an agent that has the bash tool
	agents := team.New(map[string]*agent.Agent{
		"root": agent.New("root",
			"You are a technical expert with bash capabilities.",
			agent.WithModel(llm),
			agent.WithToolSets([]tools.ToolSet{tools.NewBashTool()}),
		),
	})

	// Create a session with an initial user message
	sess := session.New(logger)
	sess.Messages = append(sess.Messages, session.UserMessage("List the files in the current directory"))

	// Create and run the runtime
	rt, err := runtime.New(logger, agents, "root")
	if err != nil {
		log.Fatal(err)
	}

	messages, err := rt.Run(ctx, sess)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(messages[len(messages)-1].Message.Content)
}
```

### Details

1. Import the tools package
2. Add the built-in bash tool with
   `agent.WithToolSets([]tools.ToolSet{tools.NewBashTool()})`
3. The agent can now execute bash commands to answer user queries

Common built-in tools include:

- `tools.NewBashTool()`: For executing bash commands
- `tools.TaskTool{}`: For task management
- `&tools.ThinkTool{}`: For metacognitive reasoning

## Problem: Creating Custom Tools

You need to create a custom tool for specialized functionality not covered by
the built-in tools.

### Solution

```go
// Define a handler function for our custom tool
func addNumbers(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	// Define the parameters structure
	type params struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	// Parse the parameters from the tool call
	var p params
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &p); err != nil {
		return nil, err
	}

	// Return the result
	return &tools.ToolCallResult{
		Output: fmt.Sprintf("%d", p.A+p.B),
	}, nil
}

// Create an agent with the custom tool
agent := agent.New(
	"root",
	"You are a math assistant with calculation abilities.",
	agent.WithModel(llm),
	agent.WithTools([]tools.Tool{
		{
			Handler: addNumbers,
			Function: &tools.FunctionDefinition{
				Name:        "add",
				Description: "Add two numbers",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"a": map[string]any{
							"type": "number",
						},
						"b": map[string]any{
							"type": "number",
						},
					},
				},
			},
		},
	}),
)
```

### Complete Example

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/model/provider/openai"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/rumpl/cagent/pkg/team"
	"github.com/rumpl/cagent/pkg/tools"
)

// Define a handler function for our custom tool
func addNumbers(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	// Define the parameters structure
	type params struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	// Parse the parameters from the tool call
	var p params
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &p); err != nil {
		return nil, err
	}

	fmt.Println("Adding numbers", p.A, p.B)

	// Return the result
	return &tools.ToolCallResult{
		Output: fmt.Sprintf("%d", p.A+p.B),
	}, nil
}

func main() {
	ctx := context.Background()
	logger := slog.Default()

	// Create a language model client
	llm, err := openai.NewClient(&config.ModelConfig{
		Type:  "openai",
		Model: "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a team with an agent that has our custom tool
	agents := team.New(map[string]*agent.Agent{
		"root": agent.New("root",
			"You are a math assistant with calculation abilities.",
			agent.WithModel(llm),
			agent.WithTools([]tools.Tool{
				{
					Handler: addNumbers,
					Function: &tools.FunctionDefinition{
						Name:        "add",
						Description: "Add two numbers",
						Parameters: tools.FunctionParamaters{
							Type: "object",
							Properties: map[string]any{
								"a": map[string]any{
									"type": "number",
								},
								"b": map[string]any{
									"type": "number",
								},
							},
						},
					},
				},
			}),
		),
	})

	// Create a session with an initial user message
	sess := session.New(logger)
	sess.Messages = append(sess.Messages, session.UserMessage("What is 1 + 2?"))

	// Create and run the runtime
	rt, err := runtime.New(logger, agents, "root")
	if err != nil {
		log.Fatal(err)
	}

	messages, err := rt.Run(ctx, sess)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(messages[len(messages)-1].Message.Content)
}
```

### Details

1. Create a handler function that processes the tool call and returns a result
2. Define the expected parameters using a struct with JSON tags
3. Parse the incoming parameters using `json.Unmarshal`
4. Return the result as a `ToolCallResult`
5. Register the tool with the agent using `agent.WithTools`
6. Define the tool interface with name, description, and parameter schema

## Problem: Combining Multiple Features

You need to create a sophisticated agent system that combines multiple features:
delegation, tools, and specialized agents.

### Solution

```go
// Create a database specialist with database tools
dbSpecialist := agent.New(
	"database",
	"You are a database specialist with SQL expertise.",
	agent.WithModel(llm),
	agent.WithToolSets([]tools.ToolSet{sqliteTool})
)

// Create a file specialist with filesystem tools
fileSpecialist := agent.New(
	"filesystem",
	"You are a file system specialist who can manage files and directories.",
	agent.WithModel(llm),
	agent.WithToolSets([]tools.ToolSet{filesystemTool})
)

// Create a coordinator that can delegate to specialists
coordinator := agent.New(
	"coordinator",
	"You are a coordinator agent. Delegate database tasks to the database specialist and file tasks to the filesystem specialist.",
	agent.WithModel(llm),
	agent.WithSubAgents([]*agent.Agent{dbSpecialist, fileSpecialist}),
	agent.WithToolSets([]tools.ToolSet{&tools.ThinkTool{}})
)

// Create the team with all agents
agents := team.New(map[string]*agent.Agent{
	"coordinator": coordinator,
	"database": dbSpecialist,
	"filesystem": fileSpecialist,
})
```

### Details

1. Create specialized agents with appropriate tools for their domain
2. Create a coordinator agent that can delegate to the specialists
3. Give the coordinator a think tool for metacognitive reasoning
4. Add all agents to the team
5. Start with the coordinator agent in the runtime

This approach creates a flexible system where:

- The coordinator handles initial requests and decides which specialist to use
- Specialists have domain-specific tools and knowledge
- The coordinator can think through complex problems before delegating

## Problem: Running Multiple Conversations

You need to run multiple conversations with the same agent team.

### Solution

```go
// Create the agent team once
agents := team.New(map[string]*agent.Agent{
	"root": agent.New("root", "You are a helpful assistant.", agent.WithModel(llm)),
})

// Create the runtime
rt, err := runtime.New(logger, agents, "root")
if err != nil {
	log.Fatal(err)
}

// First conversation
sess1 := session.New(logger)
sess1.Messages = append(sess1.Messages, session.UserMessage("What is the capital of France?"))
messages1, err := rt.Run(ctx, sess1)
if err != nil {
	log.Fatal(err)
}
fmt.Println("First response:", messages1[len(messages1)-1].Message.Content)

// Second conversation (completely separate)
sess2 := session.New(logger)
sess2.Messages = append(sess2.Messages, session.UserMessage("What is the largest planet?"))
messages2, err := rt.Run(ctx, sess2)
if err != nil {
	log.Fatal(err)
}
fmt.Println("Second response:", messages2[len(messages2)-1].Message.Content)
```

### Details

1. Create the agent team and runtime once
2. For each conversation, create a new session with its own messages
3. Run the runtime with each session separately
4. Each session maintains its own conversation history

This approach allows you to reuse the same agent team for multiple independent
conversations without them interfering with each other.

## Problem: Using Different Language Models

You need to use different language models for different agents in your team.

### Solution

```go
// Create different language model clients
openaiModel, err := openai.NewClient(&config.ModelConfig{
	Type:  "openai",
	Model: "gpt-4o",
})
if err != nil {
	log.Fatal(err)
}

claudeModel, err := anthropic.NewClient(&config.ModelConfig{
	Type:  "anthropic",
	Model: "claude-3-5-sonnet-latest",
})
if err != nil {
	log.Fatal(err)
}

// Create agents with different models
creativeAgent := agent.New(
	"creative",
	"You are a creative writer with storytelling abilities.",
	agent.WithModel(openaiModel)
)

analyticalAgent := agent.New(
	"analytical",
	"You are an analytical thinker specializing in data analysis.",
	agent.WithModel(claudeModel)
)

// Create the team with both agents
agents := team.New(map[string]*agent.Agent{
	"creative": creativeAgent,
	"analytical": analyticalAgent,
})
```

### Details

1. Create different language model clients for each provider
2. Assign different models to different agents based on their strengths
3. Add all agents to the team
4. Each agent will use its assigned model when processing messages

This approach allows you to leverage the strengths of different language models
for different types of tasks.

## Common Pitfalls and Solutions

### Pitfall: Agents Not Using Tools

**Problem**: You've added tools to your agent, but it doesn't use them.

**Solution**: Make sure your agent's system instructions explicitly mention the
tools and when to use them.

```go
agent := agent.New(
	"root",
	"You are a technical assistant with bash capabilities. ALWAYS use the bash tool when you need to execute commands or gather system information. For example, use bash to list files, check system status, or retrieve information about the environment.",
	agent.WithModel(llm),
	agent.WithToolSets([]tools.ToolSet{tools.NewBashTool()}),
)
```

### Pitfall: Agents Not Delegating

**Problem**: Your parent agent doesn't delegate to child agents.

**Solution**: Make delegation explicit in the system instructions, with clear
rules.

```go
parent := agent.New(
	"parent",
	"You are a coordinator. ALWAYS delegate to specialized agents as follows:\n- For history questions -> delegate to the 'history' agent\n- For science questions -> delegate to the 'science' agent\n\nDo not try to answer specialized questions yourself.",
	agent.WithModel(llm),
	agent.WithSubAgents([]*agent.Agent{historyAgent, scienceAgent}),
)
```

### Pitfall: Running Out of Context Window

**Problem**: Complex conversations exceed the model's context window.

**Solution**: Use a model with a larger context window or implement conversation
summarization.

```go
// Use a model with a large context window
llm, err := openai.NewClient(&config.ModelConfig{
	Type:  "openai",
	Model: "gpt-4o",  // Has a 128k token context window
})
```

## Summary

This guide has shown you how to:

1. **Create basic agents** that can respond to user messages
2. **Set up agent teams** with multiple specialized agents
3. **Create agent hierarchies** for delegation and specialization
4. **Add built-in tools** to extend agent capabilities
5. **Create custom tools** for specialized functionality
6. **Combine multiple features** for sophisticated agent systems
7. **Run multiple conversations** with the same agent team
8. **Use different language models** for different agents

By combining these techniques, you can create flexible, powerful agent systems
tailored to your specific needs.

## Related Resources

- For configuration-based approaches, see the [YAML Configuration
  Guide](./reference.md)
- For a deeper understanding of the architecture, see the [Explanation
  Guide](./explanation.md)
- For step-by-step tutorials, see the [Tutorials](./tutorial.md)
