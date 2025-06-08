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
	"github.com/rumpl/cagent/pkg/tools"
)

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

	child := agent.New("child", "You are a child, with a lot of energy.", agent.WithModel(llm))
	agents := map[string]*agent.Agent{
		"root":  agent.New("root", "You are a human, with feelings and emotions.", agent.WithModel(llm), agent.WithSubAgents([]*agent.Agent{child}), agent.WithToolSets([]tools.ToolSet{&tools.TaskTool{}})),
		"child": child,
	}

	sess := session.New(logger)
	sess.Messages = append(sess.Messages, session.UserMessage("Ask your child how they are doing and tell me what they said"))

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
