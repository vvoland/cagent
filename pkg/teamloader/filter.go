package teamloader

import (
	"context"
	"log/slog"
	"slices"

	"github.com/docker/cagent/pkg/tools"
)

func WithToolsFilter(inner tools.ToolSet, toolNames ...string) tools.ToolSet {
	if len(toolNames) == 0 {
		return inner
	}

	return &filterTools{
		ToolSet:   inner,
		toolNames: toolNames,
	}
}

type filterTools struct {
	tools.ToolSet
	toolNames []string
}

func (f *filterTools) Tools(ctx context.Context) ([]tools.Tool, error) {
	allTools, err := f.ToolSet.Tools(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []tools.Tool
	for _, tool := range allTools {
		if !slices.Contains(f.toolNames, tool.Name) {
			slog.Debug("Filtering out tool", "tool", tool.Name)
			continue
		}

		filtered = append(filtered, tool)
	}

	return filtered, nil
}
