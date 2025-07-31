package root

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/loader"
	"github.com/docker/cagent/pkg/team"
)

func loadTeams(ctx context.Context, agentsPathOrDirectory string, logger *slog.Logger) (map[string]*team.Team, error) {
	teams := make(map[string]*team.Team)

	agentPaths, err := findAgentPaths(agentsPathOrDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to find agents: %w", err)
	}

	for _, agentPath := range agentPaths {
		t, err := loader.Load(ctx, agentPath, runConfig, logger)
		if err != nil {
			logger.Warn("Failed to load agent", "file", agentPath, "error", err)
			continue
		}

		teams[t.ID] = t
	}

	return teams, nil
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
