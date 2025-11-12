package teamloader

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/config"
	latest "github.com/docker/cagent/pkg/config/v2"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/js"
	"github.com/docker/cagent/pkg/model/provider"
	"github.com/docker/cagent/pkg/model/provider/options"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tools/codemode"
)

// LoadTeams loads all agent teams from the given directory or file path
func LoadTeams(ctx context.Context, agentsPathOrDirectory string, runtimeConfig config.RuntimeConfig) (map[string]*team.Team, error) {
	teams := make(map[string]*team.Team)

	agentPaths, err := findAgentPaths(agentsPathOrDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to find agents: %w", err)
	}

	for _, agentPath := range agentPaths {
		t, err := Load(ctx, agentPath, runtimeConfig)
		if err != nil {
			slog.Warn("Failed to load agent", "file", agentPath, "error", err)
			continue
		}

		teams[t.ID] = t
	}

	return teams, nil
}

// findAgentPaths finds all agent YAML files in the given directory or returns the single file path
func findAgentPaths(agentsPathOrDirectory string) ([]string, error) {
	stat, err := os.Stat(agentsPathOrDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to stat agents path: %w", err)
	}

	if !stat.IsDir() {
		return []string{agentsPathOrDirectory}, nil
	}

	var agents []string

	agentsDirectory := agentsPathOrDirectory
	entries, err := os.ReadDir(agentsDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		agents = append(agents, filepath.Join(agentsDirectory, entry.Name()))
	}

	return agents, nil
}

type loadOptions struct {
	modelOverrides  []string
	toolsetRegistry *ToolsetRegistry
}

type Opt func(*loadOptions) error

func WithModelOverrides(overrides []string) Opt {
	return func(opts *loadOptions) error {
		opts.modelOverrides = overrides
		return nil
	}
}

// WithToolsetRegistry allows using a custom toolset registry instead of the default
func WithToolsetRegistry(registry *ToolsetRegistry) Opt {
	return func(opts *loadOptions) error {
		opts.toolsetRegistry = registry
		return nil
	}
}

// Load loads an agent team from the given file path.
// Prefers LoadFrom for more control over the source.
func Load(ctx context.Context, p string, runtimeConfig config.RuntimeConfig, opts ...Opt) (*team.Team, error) {
	return LoadFrom(ctx, NewFileSource(p), runtimeConfig, opts...)
}

// LoadFrom loads an agent team from the given source
func LoadFrom(ctx context.Context, source AgentSource, runtimeConfig config.RuntimeConfig, opts ...Opt) (*team.Team, error) {
	var loadOpts loadOptions
	loadOpts.toolsetRegistry = NewDefaultToolsetRegistry()

	for _, o := range opts {
		if err := o(&loadOpts); err != nil {
			return nil, err
		}
	}

	fileName := source.Name()
	parentDir := source.ParentDir()

	// Make env file paths absolute relative to the agent config file.
	var err error
	runtimeConfig.EnvFiles, err = environment.AbsolutePaths(parentDir, runtimeConfig.EnvFiles)
	if err != nil {
		return nil, err
	}

	envFilesProviders, err := environment.NewEnvFilesProvider(runtimeConfig.EnvFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to read env files: %w", err)
	}

	defaultEnvProvider := runtimeConfig.DefaultEnvProvider
	if defaultEnvProvider == nil {
		defaultEnvProvider = environment.NewDefaultProvider()
	}
	env := environment.NewMultiProvider(envFilesProviders, defaultEnvProvider)

	// Load the agent's configuration
	data, err := source.Read()
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg, err := config.LoadConfigBytes(data)
	if err != nil {
		return nil, err
	}

	// Apply model overrides from CLI flags before checking required env vars
	if err := config.ApplyModelOverrides(cfg, loadOpts.modelOverrides); err != nil {
		return nil, err
	}

	// Early check for required env vars before loading models and tools.
	if err := config.CheckRequiredEnvVars(ctx, cfg, env, runtimeConfig); err != nil {
		return nil, err
	}

	// Load agents
	var agents []*agent.Agent
	agentsByName := make(map[string]*agent.Agent)

	for name, agentConfig := range cfg.Agents {
		opts := []agent.Opt{
			agent.WithName(name),
			agent.WithDescription(agentConfig.Description),
			agent.WithWelcomeMessage(agentConfig.WelcomeMessage),
			agent.WithAddDate(agentConfig.AddDate),
			agent.WithAddEnvironmentInfo(agentConfig.AddEnvironmentInfo),
			agent.WithAddPromptFiles(agentConfig.AddPromptFiles),
			agent.WithMaxIterations(agentConfig.MaxIterations),
			agent.WithNumHistoryItems(agentConfig.NumHistoryItems),
			agent.WithCommands(js.Expand(ctx, agentConfig.Commands, env)),
		}

		models, err := getModelsForAgent(ctx, cfg, &agentConfig, env, runtimeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get models: %w", err)
		}
		for _, model := range models {
			opts = append(opts, agent.WithModel(model))
		}

		agentTools, warnings := getToolsForAgent(ctx, &agentConfig, parentDir, env, runtimeConfig, loadOpts.toolsetRegistry)
		if len(warnings) > 0 {
			opts = append(opts, agent.WithLoadTimeWarnings(warnings))
		}

		if len(agentConfig.SubAgents) > 0 {
			agentTools = append(agentTools, builtin.NewTransferTaskTool())
		}

		if len(agentTools) > 0 {
			opts = append(opts, agent.WithToolSets(agentTools...))
		}

		ag := agent.New(name, agentConfig.Instruction, opts...)
		agents = append(agents, ag)
		agentsByName[name] = ag
	}

	for name := range cfg.Agents {
		agentConfig := cfg.Agents[name]
		if len(agentConfig.SubAgents) == 0 {
			continue
		}

		subAgents := make([]*agent.Agent, 0, len(agentConfig.SubAgents))
		for _, subName := range agentConfig.SubAgents {
			if subAgent, exists := agentsByName[subName]; exists {
				subAgents = append(subAgents, subAgent)
			}
		}

		if a, exists := agentsByName[name]; exists && len(subAgents) > 0 {
			agent.WithSubAgents(subAgents...)(a)
		}
	}

	return team.New(team.WithID(fileName), team.WithAgents(agents...)), nil
}

func getModelsForAgent(ctx context.Context, cfg *latest.Config, a *latest.AgentConfig, env environment.Provider, runtimeConfig config.RuntimeConfig) ([]provider.Provider, error) {
	var models []provider.Provider

	for name := range strings.SplitSeq(a.Model, ",") {
		modelCfg, exists := cfg.Models[name]
		if !exists {
			return nil, fmt.Errorf("model '%s' not found in configuration", name)
		}

		opts := []options.Opt{options.WithGateway(runtimeConfig.ModelsGateway)}
		if a.StructuredOutput != nil {
			opts = append(opts, options.WithStructuredOutput(a.StructuredOutput))
		}

		model, err := provider.New(ctx, &modelCfg, env, opts...)
		if err != nil {
			return nil, err
		}

		models = append(models, model)
	}

	return models, nil
}

// getToolsForAgent returns the tool definitions for an agent based on its configuration
func getToolsForAgent(ctx context.Context, a *latest.AgentConfig, parentDir string, envProvider environment.Provider, runtimeConfig config.RuntimeConfig, registry *ToolsetRegistry) ([]tools.ToolSet, []string) {
	var (
		toolSets []tools.ToolSet
		warnings []string
	)

	for i := range a.Toolsets {
		toolset := a.Toolsets[i]

		tool, err := registry.CreateTool(ctx, toolset, parentDir, envProvider, runtimeConfig)
		if err != nil {
			// Collect error but continue loading other toolsets
			slog.Warn("Toolset configuration failed; skipping", "type", toolset.Type, "ref", toolset.Ref, "command", toolset.Command, "error", err)
			warnings = append(warnings, fmt.Sprintf("toolset %s failed: %v", toolset.Type, err))
			continue
		}

		wrapped := WithToolsFilter(tool, toolset.Tools...)
		wrapped = WithInstructions(wrapped, toolset.Instruction)
		wrapped = WithToon(wrapped, toolset.Toon)

		toolSets = append(toolSets, wrapped)
	}

	// Wrap all tools in a single Code Mode toolset.
	// This allows the agent to call multiple tools in a single response.
	// It also allows to combine the results of multiple tools in a single response.
	if a.CodeModeTools || runtimeConfig.GlobalCodeMode {
		toolSets = []tools.ToolSet{codemode.Wrap(toolSets...)}
	}

	return toolSets, warnings
}
