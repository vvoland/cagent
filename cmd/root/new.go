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

var modelProvider string
var modelName string

// Cmd creates a new command to create a new agent configuration
func NewNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new agent configuration",
		Long:  `Create a new agent configuration by asking questions and generating a YAML file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := slog.Default()

			// Validate flag combinations immediately: --model requires explicit --provider
			flags := cmd.Flags()
			if flags.Changed("model") && !flags.Changed("provider") {
				return fmt.Errorf("--model can only be used together with --provider")
			}

			// Auto-select provider if none explicitly provided (before any prompts)
			if !flags.Changed("provider") {
				if runConfig.ModelsGateway == "" {
					// Prefer Anthropic, then OpenAI, then Google based on available API keys
					switch {
					case os.Getenv("ANTHROPIC_API_KEY") != "":
						modelProvider = "anthropic"
					case os.Getenv("OPENAI_API_KEY") != "":
						modelProvider = "openai"
					case os.Getenv("GOOGLE_API_KEY") != "":
						modelProvider = "google"
					default:
						return fmt.Errorf("no provider credentials found; set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_API_KEY, or pass --provider")
					}
				} else {
					// Using Models Gateway; default to Anthropic if not specified
					modelProvider = "anthropic"
				}
			}

			prompt := ""
			if len(args) > 0 {
				prompt = strings.Join(args, " ")
			} else {
				reader := bufio.NewReader(os.Stdin)

				fmt.Printf("%s\n", blue("------- Welcome to %s! -------", bold(APP_NAME)))
				fmt.Printf("%s\n\n", gray("         (Ctrl+C to exit)"))
				fmt.Printf("%s\n\n", blue("What should your agent/agent team do? (describe its purpose)"))
				fmt.Print(blue("> "))

				var err error
				prompt, err = reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read purpose: %w", err)
				}
				prompt = strings.TrimSpace(prompt)
				fmt.Println()
			}

			out, err := creator.StreamCreateAgent(ctx, ".", logger, prompt, runConfig, modelProvider, modelName)
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
	cmd.PersistentFlags().StringVar(&modelProvider, "provider", "", "Model provider to use: anthropic, openai, google. if not specified, they will be picked in that order based on available API keys")
	cmd.PersistentFlags().StringVar(&modelName, "model", "", "Model to use (overrides provider default)")

	return cmd
}
