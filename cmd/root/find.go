package root

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/loader"
	"github.com/docker/cagent/pkg/runtime"
)

func loadAgents(ctx context.Context, agentsPathOrDirectory string, logger *slog.Logger) (map[string]*runtime.Runtime, error) {
	runtimes := make(map[string]*runtime.Runtime)

	agentPaths, err := findAgentPaths(agentsPathOrDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to find agents: %w", err)
	}

	for _, agentPath := range agentPaths {
		team, err := loader.Load(ctx, agentPath, runConfig, logger)
		if err != nil {
			logger.Warn("Failed to load agent", "file", agentPath, "error", err)
			continue
		}

		runtimes[filepath.Base(agentPath)] = runtime.New(logger, team, "root")
	}

	return runtimes, nil
}

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
