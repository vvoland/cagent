package root

import (
	"context"
	"strings"

	"github.com/docker/cagent/pkg/aliases"
	"github.com/docker/cagent/pkg/config"
	"github.com/spf13/cobra"
)

func completeRunExec(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeAlias(toComplete)
	case 1:
		return completeMessage(cmd, args, toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeAlias(toComplete string) ([]string, cobra.ShellCompDirective) {
	if strings.Contains(toComplete, "/") || strings.HasPrefix(toComplete, ".") {
		return nil, cobra.ShellCompDirectiveDefault
	}

	s, err := aliases.Load()
	if err != nil {
		// Ignore error and don't provide alias completions
		return nil, cobra.ShellCompDirectiveDefault
	}

	var candidates []string
	for k, v := range s.List() {
		if strings.HasPrefix(k, toComplete) {
			candidates = append(candidates, k+"\t"+v)
		}
	}

	return candidates, cobra.ShellCompDirectiveDefault
}

func completeMessage(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	agentSource, err := config.Resolve(args[0])
	if err != nil {
		// Ignore error and don't provide completions
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.Load(context.Background(), agentSource)
	if err != nil {
		// Ignore error and don't provide completions
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	agent := "root"
	name, err := cmd.Flags().GetString("agent")
	if err == nil && name != "" {
		agent = name
	}

	var candidates []string
	for k, v := range cfg.Agents[agent].Commands {
		if strings.HasPrefix(k, toComplete) {
			candidates = append(candidates, "/"+k+"\t"+v)
		}
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}
