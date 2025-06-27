package loader

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/mcp"
	"github.com/rumpl/cagent/pkg/memory"
	"github.com/rumpl/cagent/pkg/memory/database/sqlite"
	"github.com/rumpl/cagent/pkg/model/provider"
	"github.com/rumpl/cagent/pkg/team"
	"github.com/rumpl/cagent/pkg/tools"
)

func Agents(ctx context.Context, path string, logger *slog.Logger) (*team.Team, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}

	fac := provider.NewFactory()

	agents := make(map[string]*agent.Agent)
	for name, agentConfig := range cfg.Agents {
		modelCfg, exists := cfg.Models[agentConfig.Model]
		if !exists {
			return nil, fmt.Errorf("model '%s' not found in configuration", agentConfig.Model)
		}

		model, err := fac.NewProvider(&modelCfg)
		if err != nil {
			return nil, err
		}
		opts := []agent.Opt{
			agent.WithName(name),
			agent.WithModel(model),
			agent.WithDescription(agentConfig.Description),
			agent.WithAddDate(agentConfig.AddDate),
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
			mm := memory.NewManager(db, model)
			opts = append(opts, agent.WithMemoryManager(mm))
			agentTools = append(agentTools, tools.NewMemoryTool(mm))
		}

		opts = append(opts, agent.WithToolSets(agentTools))

		agents[name] = agent.New(name, agentConfig.Instruction, opts...)
	}

	for name, agentConfig := range cfg.Agents {
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

// getToolsForAgent returns the tool definitions for an agent based on its configuration
func getToolsForAgent(ctx context.Context, a *config.AgentConfig, logger *slog.Logger) ([]tools.ToolSet, error) {
	var t []tools.ToolSet

	if len(a.SubAgents) > 0 {
		t = append(t, tools.NewTaskTool())
	}

	if a.Think {
		t = append(t, tools.NewThinkTool())
	}

	if a.Todo {
		t = append(t, tools.NewTodoTool())
	}

	toolsets := a.Toolsets
	for _, toolset := range toolsets {
		// TODO: we will have more builtin tools in the future
		if toolset.Type == "builtin" {
			t = append(t, tools.NewBashTool())
		}
		if toolset.Type != "mcp" {
			continue
		}

		envSlice := make([]string, 0, len(toolset.Env))
		for k, v := range toolset.Env {
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
