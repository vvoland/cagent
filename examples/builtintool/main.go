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
	"github.com/rumpl/cagent/pkg/tools/builtin"
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
			agent.WithToolSets([]tools.ToolSet{builtin.NewBashTool()}),
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
