package root

import (
	"strings"

	"github.com/docker/cagent/pkg/aliases"
	"github.com/spf13/cobra"
)

func completeRunExec(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if strings.Contains(toComplete, "/") || strings.HasPrefix(toComplete, ".") {
		return nil, cobra.ShellCompDirectiveDefault
	}

	s, err := aliases.Load()
	if err != nil {
		// Ignore error and don't provide completions
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
