package main

import (
	"context"
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
	"github.com/docker/cagent/pkg/tools/builtin"
)

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
			"You are an expert hacker",
			agent.WithModel(llm),
			agent.WithToolSets([]tools.ToolSet{builtin.NewShellTool()}),
		),
	})

	sess := session.New(logger, session.WithUserMessage("Tell me a story about my current directory"))

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
