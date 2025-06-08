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

	agents := map[string]*agent.Agent{
		"root": agent.New("root",
			"You are an expert hacker",
			agent.WithModel(llm),
			agent.WithToolSets([]tools.ToolSet{tools.NewBashTool()}),
		),
	}

	rt, err := runtime.New(logger, agents, "root")
	if err != nil {
		log.Fatal(err)
	}

	sess := session.New(logger)
	sess.Messages = append(sess.Messages, session.UserMessage("Tell me a story about my current directory"))

	messages, err := rt.Run(ctx, sess)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(messages[len(messages)-1].Message.Content)
}
