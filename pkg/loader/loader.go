package loader

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/mcp"
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
		model, err := fac.NewProviderFromConfig(cfg, agentConfig.Model)
		if err != nil {
			return nil, err
		}
		opts := []agent.Opt{
			agent.WithName(name),
			agent.WithModel(model),
			agent.WithDescription(agentConfig.Description),
			agent.WithAddDate(agentConfig.AddDate),
		}

		agentTools, err := getToolsForAgent(ctx, cfg, name, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to get tools: %w", err)
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
func getToolsForAgent(ctx context.Context, cfg *config.Config, agentName string, logger *slog.Logger) ([]tools.ToolSet, error) {
	a, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found in configuration", agentName)
	}

	var t []tools.ToolSet

	if len(a.SubAgents) > 0 {
		if cfg.Type == "task" {
			t = append(t, tools.NewTaskTool())
		} else {
			t = append(t, tools.NewAgentTransferTool())
		}
	}

	if a.Think {
		t = append(t, tools.NewThinkTool())
	}

	if a.Todo {
		t = append(t, tools.NewTodoTool())
	}

	toolsets := a.Toolsets
	for _, toolset := range toolsets {
		if toolset.Type == "builtin" {
			t = append(t, tools.NewBashTool())
		}
		if toolset.Type != "mcp" {
			continue
		}

		// Convert env map to string slice
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
