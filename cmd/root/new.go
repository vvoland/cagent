package root

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/internal/creator"
	"github.com/docker/cagent/pkg/runtime"
)

// Cmd creates a new command to create a new agent configuration
func NewNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new agent configuration",
		Long:  `Create a new agent configuration by asking questions and generating a YAML file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := slog.Default()

			prompt := ""
			if len(args) > 0 {
				prompt = strings.Join(args, " ")
			} else {
				reader := bufio.NewReader(os.Stdin)

				fmt.Print(blue("Welcome to %s! (Ctrl+C to exit)\n\nWhat should your agent/agent team do? (describe its purpose):\n\n> ", bold(APP_NAME)))
				var err error
				prompt, err = reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read purpose: %w", err)
				}
				prompt = strings.TrimSpace(prompt)
				fmt.Println()
			}

			out, err := creator.StreamCreateAgent(ctx, ".", logger, prompt, runConfig)
			if err != nil {
				return err
			}

			llmIsTyping := false

			for event := range out {
				switch e := event.(type) {
				case *runtime.AgentChoiceEvent:
					if !llmIsTyping {
						fmt.Println()
						llmIsTyping = true
					}
					fmt.Printf("%s", e.Choice.Delta.Content)
				case *runtime.ToolCallEvent:
					if llmIsTyping {
						fmt.Println()
						llmIsTyping = false
					}
					printToolCall(e.ToolCall)
				case *runtime.ToolCallResponseEvent:
					if llmIsTyping {
						fmt.Println()
						llmIsTyping = false
					}
					printToolCallResponse(e.ToolCall, e.Response)
				case *runtime.ErrorEvent:
					if llmIsTyping {
						fmt.Println()
						llmIsTyping = false
					}
					printError(fmt.Errorf("%s", e.Error))
				}
			}
			fmt.Print("\n\n")
			return nil
		},
	}
	addGatewayFlags(cmd)

	return cmd
}
