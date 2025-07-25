package root

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
