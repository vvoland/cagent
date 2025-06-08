package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/model/provider/openai"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/rumpl/cagent/pkg/tools"
)

type tool struct {
}

func (t *tool) Tools(ctx context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Type:    "function",
			Handler: t,
			Function: &tools.FunctionDefinition{
				Name:        "add",
				Description: "Add two numbers",
				Parameters: mcp.ToolInputSchema{
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
	}, nil
}

func (t *tool) CallTool(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	type params struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	var p params
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &p); err != nil {
		return nil, err
	}

	return &tools.ToolCallResult{
		Output: fmt.Sprintf("%d", p.A+p.B),
	}, nil
}

func (t *tool) Instructions() string {
	return ""
}

func (t *tool) Start(ctx context.Context) error {
	return nil
}

func (t *tool) Stop() error {
	return nil
}

func main() {
	ctx := context.Background()
	logger := slog.Default()

	llm, err := openai.NewClient(&config.ModelConfig{
		Type:  "openai",
		Model: "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	agents := map[string]*agent.Agent{
		"root": agent.New("root",
			"You are a human, with feelings and emotions.",
			agent.WithModel(llm),
			agent.WithToolSets([]tools.ToolSet{&tool{}})),
	}

	rt, err := runtime.New(logger, agents, "root")
	if err != nil {
		log.Fatal(err)
	}

	sess := session.New(logger)
	sess.Messages = append(sess.Messages, session.UserMessage("What is 1 + 2?"))

	messages, err := rt.Run(ctx, sess)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(messages[len(messages)-1].Message.Content)
}
