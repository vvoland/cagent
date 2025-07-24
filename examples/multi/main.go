package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/environment"
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
	}, environment.NewDefaultProvider(logger), logger)
	if err != nil {
		log.Fatal(err)
	}

	child := agent.New("child",
		"You are a child, with a lot of energy.",
		agent.WithModel(llm),
		agent.WithDescription("A child."),
	)
	root := agent.New("root",
		"You are a human, with feelings and emotions.",
		agent.WithModel(llm),
		agent.WithSubAgents([]*agent.Agent{child}),
		agent.WithToolSets([]tools.ToolSet{&builtin.TransferTaskTool{}}),
	)

	agents := team.New(map[string]*agent.Agent{
		"root":  root,
		"child": child,
	})

	sess := session.New(logger, session.WithUserMessage("Ask your child how they are doing and tell me what they said"))

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
