package creator

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/agent"
	latest "github.com/docker/cagent/pkg/config/v1"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider"
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

func CreateAgent(ctx context.Context, baseDir, prompt string, runConfig latest.RuntimeConfig) (out, path string, err error) {
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
	rt, err := runtime.New(newTeam)
	if err != nil {
		return "", "", fmt.Errorf("failed to create runtime: %w", err)
	}

	sess := session.New(session.WithUserMessage("", prompt))
	sess.ToolsApproved = true

	messages, err := rt.Run(ctx, sess)
	if err != nil {
		return "", "", fmt.Errorf("failed to run session: %w", err)
	}

	return messages[len(messages)-1].Message.Content, fsToolset.path, nil
}

func StreamCreateAgent(ctx context.Context, baseDir, prompt string, runConfig latest.RuntimeConfig, providerName, modelNameOverride string) (<-chan runtime.Event, error) {
	// Select default model per provider
	prov := providerName

	var modelName string
	switch prov {
	case "openai":
		modelName = "gpt-5-mini"
	case "google":
		modelName = "gemini-2.5-flash"
	case "anthropic", "":
		prov = "anthropic"
		modelName = "claude-sonnet-4-0"
	default:
		// Fallback to anthropic if unknown
		prov = "anthropic"
		modelName = "claude-sonnet-4-0"
	}

	// If a specific model override is provided, use it
	if modelNameOverride != "" {
		modelName = modelNameOverride
	}

	// If not using a models gateway, avoid selecting a provider the user can't run
	usableProviders := []string{}
	if runConfig.ModelsGateway == "" {
		if os.Getenv("OPENAI_API_KEY") != "" {
			usableProviders = append(usableProviders, "openai")
		}
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			usableProviders = append(usableProviders, "anthropic")
		}
		if os.Getenv("GOOGLE_API_KEY") != "" {
			usableProviders = append(usableProviders, "google")
		}
	}
	modelsPerProvider := map[string]string{
		"openai":    "gpt-5-mini",
		"anthropic": "claude-sonnet-4-0",
		"google":    "gemini-2.5-flash",
	}

	llm, err := provider.New(
		ctx,
		&latest.ModelConfig{
			Provider:  prov,
			Model:     modelName,
			MaxTokens: 64000,
		},
		environment.NewDefaultProvider(),
		options.WithGateway(runConfig.ModelsGateway),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	fmt.Println("Generating agent configuration....")

	fsToolset := fsToolset{inner: builtin.NewFilesystemTool([]string{baseDir})}
	fileName := filepath.Base(fsToolset.path)

	// Provide soft guidance to prefer the selected providers
	instructions := agentBuilderInstructions + "\n\nPreferred model providers to use: " + strings.Join(usableProviders, ", ") + ". You must always use one or more of the following model configurations: \n"
	for _, provider := range usableProviders {
		instructions += fmt.Sprintf(`
		version: "1"
		models:
			%s:
				type: %s
				model: %s
				max_tokens: 64000\n`, provider, provider, modelsPerProvider[provider])
	}

	newTeam := team.New(
		team.WithID(fileName),
		team.WithAgents(
			agent.New(
				"root",
				instructions,
				agent.WithModel(llm),
				agent.WithToolSets(
					builtin.NewShellTool(),
					&fsToolset,
				),
			)))
	rt, err := runtime.New(newTeam)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime: %w", err)
	}

	sess := session.New(session.WithUserMessage("", prompt))
	sess.ToolsApproved = true

	return rt.RunStream(ctx, sess), nil
}
