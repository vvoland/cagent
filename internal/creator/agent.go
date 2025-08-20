package creator

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/docker/cagent/pkg/agent"
	latest "github.com/docker/cagent/pkg/config/v1"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider/anthropic"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
)

//go:embed instructions.txt
var agentBuilderInstructions string

type fsToolset struct {
	inner                    tools.ToolSet
	originalWriteFileHandler tools.ToolHandler
	path                     string
}

func (f *fsToolset) Instructions() string {
	return f.inner.Instructions()
}

func (f *fsToolset) Start(ctx context.Context) error {
	return f.inner.Start(ctx)
}

func (f *fsToolset) Stop() error {
	return f.inner.Stop()
}

func (f *fsToolset) Tools(ctx context.Context) ([]tools.Tool, error) {
	innerTools, err := f.inner.Tools(ctx)
	if err != nil {
		return nil, err
	}

	for i, tool := range innerTools {
		if tool.Function != nil && tool.Function.Name == "write_file" {
			f.originalWriteFileHandler = tool.Handler
			innerTools[i].Handler = f.customWriteFileHandler
		}
	}

	return innerTools, nil
}

func (f *fsToolset) customWriteFileHandler(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	f.path = args.Path

	return f.originalWriteFileHandler(ctx, toolCall)
}

func CreateAgent(ctx context.Context, baseDir string, logger *slog.Logger, prompt string, runConfig latest.RuntimeConfig) (out, path string, err error) {
	llm, err := anthropic.NewClient(
		ctx,
		&latest.ModelConfig{
			Provider:  "anthropic",
			Model:     "claude-sonnet-4-0",
			MaxTokens: 64000,
		},
		environment.NewOsEnvProvider(),
		logger,
		options.WithGateway(runConfig.ModelsGateway),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create LLM client: %w", err)
	}

	fmt.Println("Generating agent configuration....")

	fsToolset := fsToolset{inner: builtin.NewFilesystemTool([]string{baseDir})}
	fileName := filepath.Base(fsToolset.path)
	newTeam := team.New(
		team.WithID(fileName),
		team.WithAgents(
			agent.New(
				"root",
				agentBuilderInstructions,
				agent.WithModel(llm),
				agent.WithToolSets(
					builtin.NewShellTool(),
					&fsToolset,
				),
			)))
	rt := runtime.New(logger, newTeam)

	sess := session.New(logger, session.WithUserMessage("", prompt))
	sess.ToolsApproved = true

	messages, err := rt.Run(ctx, sess)
	if err != nil {
		return "", "", fmt.Errorf("failed to run session: %w", err)
	}

	return messages[len(messages)-1].Message.Content, fsToolset.path, nil
}

func StreamCreateAgent(ctx context.Context, baseDir string, logger *slog.Logger, prompt string, runConfig latest.RuntimeConfig) (<-chan runtime.Event, error) {
	llm, err := anthropic.NewClient(
		ctx,
		&latest.ModelConfig{
			Provider:  "anthropic",
			Model:     "claude-sonnet-4-0",
			MaxTokens: 64000,
		},
		environment.NewOsEnvProvider(),
		logger,
		options.WithGateway(runConfig.ModelsGateway),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	fmt.Println("Generating agent configuration....")

	fsToolset := fsToolset{inner: builtin.NewFilesystemTool([]string{baseDir})}
	fileName := filepath.Base(fsToolset.path)
	newTeam := team.New(
		team.WithID(fileName),
		team.WithAgents(
			agent.New(
				"root",
				agentBuilderInstructions,
				agent.WithModel(llm),
				agent.WithToolSets(
					builtin.NewShellTool(),
					&fsToolset,
				),
			)))
	rt := runtime.New(logger, newTeam)

	sess := session.New(logger, session.WithUserMessage("", prompt))
	sess.ToolsApproved = true

	return rt.RunStream(ctx, sess), nil
}
