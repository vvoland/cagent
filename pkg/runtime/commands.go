package runtime

import (
	"context"
	"strings"
)

func ResolveCommand(ctx context.Context, rt Runtime, userInput string) string {
	if !strings.HasPrefix(userInput, "/") {
		return userInput
	}

	cmd, rest, _ := strings.Cut(userInput, " ")
	prompt, found := rt.CurrentAgentCommands(ctx)[cmd[1:]]
	if found {
		userInput = prompt
		if rest != "" {
			userInput += " " + rest
		}
	}

	return userInput
}
