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

func loadAgents(ctx context.Context, agentsPath string, logger *slog.Logger) (map[string]*runtime.Runtime, error) {
	runtimes := make(map[string]*runtime.Runtime)

	agents, err := findAgents(agentsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find agents: %w", err)
	}

	for _, agentPath := range agents {
		fileTeam, err := loader.Load(ctx, agentPath, envFiles, gateway, logger)
		if err != nil {
			logger.Warn("Failed to load agent", "file", agentPath, "error", err)
			continue
		}

		filename := filepath.Base(agentPath)
		rt, err := runtime.New(logger, fileTeam, "root")
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime for file %s: %w", filename, err)
		}
		runtimes[filename] = rt
	}

	return runtimes, nil
}

func findAgents(agentsPath string) ([]string, error) {
	stat, err := os.Stat(agentsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat agents path: %w", err)
	}

	if !stat.IsDir() {
		return []string{agentsPath}, nil
	}

	var agents []string

	entries, err := os.ReadDir(agentsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		agents = append(agents, filepath.Join(agentsPath, entry.Name()))
	}

	return agents, nil
}
