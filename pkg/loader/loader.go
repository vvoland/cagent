package loader

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/memory"
	"github.com/docker/cagent/pkg/memory/database/sqlite"
	"github.com/docker/cagent/pkg/model/provider"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tools/mcp"
)

func Load(ctx context.Context, path string, logger *slog.Logger) (*team.Team, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}

	agents := make(map[string]*agent.Agent)
	for name := range cfg.Agents {
		agentConfig := cfg.Agents[name]

		opts := []agent.Opt{
			agent.WithName(name),
			agent.WithDescription(agentConfig.Description),
			agent.WithAddDate(agentConfig.AddDate),
		}
		models, err := getModelsForAgent(cfg, &agentConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to get models: %w", err)
		}
		for _, model := range models {
			opts = append(opts, agent.WithModel(model))
		}

		a, ok := cfg.Agents[name]
		if !ok {
			return nil, fmt.Errorf("agent '%s' not found in configuration", name)
		}
		agentTools, err := getToolsForAgent(ctx, &a, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to get tools: %w", err)
		}

		if a.MemoryConfig.Path != "" {
			db, err := sqlite.NewMemoryDatabase(a.MemoryConfig.Path)
			if err != nil {
				return nil, fmt.Errorf("failed to create memory database: %w", err)
			}
			mm := memory.NewManager(db, models[0])
			opts = append(opts, agent.WithMemoryManager(mm))
			agentTools = append(agentTools, builtin.NewMemoryTool(mm))
		}

		opts = append(opts, agent.WithToolSets(agentTools))

		agents[name] = agent.New(name, agentConfig.Instruction, opts...)
	}

	for name := range cfg.Agents {
		agentConfig := cfg.Agents[name]
		if len(agentConfig.SubAgents) > 0 {
			subAgents := make([]*agent.Agent, 0, len(agentConfig.SubAgents))
			for _, subName := range agentConfig.SubAgents {
				if subAgent, exists := agents[subName]; exists {
					subAgents = append(subAgents, subAgent)
				}
			}

			if a, exists := agents[name]; exists && len(subAgents) > 0 {
				agent.WithSubAgents(subAgents)(a)
			}
		}
	}

	return team.New(agents), nil
}

func getModelsForAgent(cfg *config.Config, a *config.AgentConfig, logger *slog.Logger) ([]provider.Provider, error) {
	modelNames := strings.Split(a.Model, ",")
	models := make([]provider.Provider, 0, len(modelNames))
	for _, modelName := range modelNames {
		modelCfg, exists := cfg.Models[modelName]
		if !exists {
			return nil, fmt.Errorf("model '%s' not found in configuration", modelName)
		}

		model, err := provider.New(&modelCfg, logger)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

// getToolsForAgent returns the tool definitions for an agent based on its configuration
func getToolsForAgent(ctx context.Context, a *config.AgentConfig, logger *slog.Logger) ([]tools.ToolSet, error) {
	var t []tools.ToolSet

	if len(a.SubAgents) > 0 {
		t = append(t, builtin.NewTransferTaskTool())
	}

	if a.Think {
		t = append(t, builtin.NewThinkTool())
	}

	if a.Todo {
		t = append(t, builtin.NewTodoTool())
	}

	toolsets := a.Toolsets
	for _, toolset := range toolsets {
		if toolset.Type == "shell" {
			t = append(t, builtin.NewShellTool())
		}
		if toolset.Type != "mcp" {
			continue
		}

		envSlice := make([]string, 0, len(toolset.Env))
		for k, v := range toolset.Env {
			if after, ok := strings.CutPrefix(v, "$"); ok {
				envVar := after
				v = os.Getenv(envVar)
			}
			envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
		}
		mcpc, err := mcp.NewToolset(ctx, toolset.Command, toolset.Args, envSlice, toolset.Tools, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create mcp client: %w", err)
		}

		if err := mcpc.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start mcp client: %w", err)
		}

		t = append(t, mcpc)
	}
	return t, nil
}
