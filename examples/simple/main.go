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

	agents := team.New(
		agent.New(
			"root",
			"You are a human, with feelings and emotions.",
			agent.WithModel(llm),
			agent.WithDescription("A human."),
		))
	rt := runtime.New(logger, agents)

	sess := session.New(logger, session.WithUserMessage("", "How are you doing?"))

	messages, err := rt.Run(ctx, sess)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(messages[len(messages)-1].Message.Content)
}
