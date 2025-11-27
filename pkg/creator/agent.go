package creator

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider/anthropic"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/teamloader"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
)

//go:embed instructions.txt
var agentBuilderInstructions string

type fsToolset struct {
	tools.ElicitationTool

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

func (f *fsToolset) Stop(ctx context.Context) error {
	return f.inner.Stop(ctx)
}

func (f *fsToolset) Tools(ctx context.Context) ([]tools.Tool, error) {
	innerTools, err := f.inner.Tools(ctx)
	if err != nil {
		return nil, err
	}

	for i, tool := range innerTools {
		if tool.Name == builtin.ToolNameWriteFile {
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

func CreateAgent(ctx context.Context, baseDir, prompt string, runConfig *config.RuntimeConfig) (out, path string, err error) {
	llm, err := anthropic.NewClient(
		ctx,
		&latest.ModelConfig{
			Provider:  "anthropic",
			Model:     "claude-sonnet-4-0",
			MaxTokens: 64000,
		},
		environment.NewDefaultProvider(),
		options.WithGateway(runConfig.ModelsGateway),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create LLM client: %w", err)
	}

	slog.Info("Generating agent configuration....")

	fsToolset := fsToolset{inner: builtin.NewFilesystemTool([]string{baseDir})}
	newTeam := team.New(
		team.WithAgents(
			agent.New(
				"root",
				agentBuilderInstructions,
				agent.WithModel(llm),
				agent.WithToolSets(
					builtin.NewShellTool(os.Environ(), runConfig),
					&fsToolset,
				),
			)))
	rt, err := runtime.New(newTeam)
	if err != nil {
		return "", "", fmt.Errorf("failed to create runtime: %w", err)
	}

	sess := session.New(
		session.WithUserMessage(prompt),
		session.WithToolsApproved(true),
	)

	messages, err := rt.Run(ctx, sess)
	if err != nil {
		return "", "", fmt.Errorf("failed to run session: %w", err)
	}

	return messages[len(messages)-1].Message.Content, fsToolset.path, nil
}

func Agent(ctx context.Context, runConfig *config.RuntimeConfig, modelNameOverride string) (*team.Team, error) {
	usableProviders := config.AvailableProviders(ctx, runConfig.ModelsGateway, runConfig.EnvProvider())

	// Provide soft guidance to prefer the selected providers
	instructions := agentBuilderInstructions
	instructions += "\n\nPreferred model providers to use: " + strings.Join(usableProviders, ", ")
	instructions += ". You must always use one or more of the following model configurations: \n"

	for _, provider := range usableProviders {
		model := config.DefaultModels[provider]
		maxTokens := config.PreferredMaxTokens(provider)
		instructions += fmt.Sprintf(`
		models:
			%s:
				provider: %s
				model: %s
				max_tokens: %d\n`, provider, provider, model, maxTokens)
	}

	// Define a new agent configuration
	newAgentConfig := latest.Config{
		Agents: map[string]latest.AgentConfig{
			"root": {
				WelcomeMessage: `Hello! I'm here to create agents for you.

Can you explain to me what the agent will be used for?`,
				Instruction: instructions,
				Model:       "auto",
				Toolsets: []latest.Toolset{
					{Type: "shell"},
					{Type: "filesystem"},
				},
			},
		},
	}

	configAsJSON, err := json.Marshal(newAgentConfig)
	if err != nil {
		return nil, fmt.Errorf("marshalling config: %w", err)
	}

	// Custom tool registry to include fsToolset
	fsToolset := fsToolset{
		inner: builtin.NewFilesystemTool([]string{runConfig.WorkingDir}),
	}

	registry := teamloader.NewDefaultToolsetRegistry()
	registry.Register("filesystem", func(context.Context, latest.Toolset, string, *config.RuntimeConfig) (tools.ToolSet, error) {
		return &fsToolset, nil
	})

	return teamloader.Load(
		ctx,
		agentfile.NewBytesSource(configAsJSON),
		runConfig,
		teamloader.WithModelOverrides([]string{modelNameOverride}),
		teamloader.WithToolsetRegistry(registry),
	)
}
