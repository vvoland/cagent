package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/model/provider/openai"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/tools"
)

func addNumbers(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	type params struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	var p params
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &p); err != nil {
		return nil, err
	}

	fmt.Println("Adding numbers", p.A, p.B)

	return &tools.ToolCallResult{
		Output: fmt.Sprintf("%d", p.A+p.B),
	}, nil
}

func main() {
	ctx := context.Background()
	logger := slog.Default()

	llm, err := openai.NewClient(&config.ModelConfig{
		Type:  "openai",
		Model: "gpt-4o",
	}, logger)
	if err != nil {
		log.Fatal(err)
	}

	agents := team.New(map[string]*agent.Agent{
		"root": agent.New("root",
			"You are a human, with feelings and emotions.",
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

	sess := session.New(logger, session.WithUserMessage("What is 1 + 2?"))

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
