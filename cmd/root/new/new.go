package new

import (
	"bufio"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider/anthropic"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
)

//go:embed instructions.txt
var agentBuilderInstructions string

// Cmd creates a new command to create a new agent configuration
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new agent configuration",
		Long:  `Create a new agent configuration by asking questions and generating a YAML file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := slog.Default()

			reader := bufio.NewReader(os.Stdin)

			fmt.Print("What should your agent do? (describe its purpose): ")
			prompt, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read purpose: %w", err)
			}
			prompt = strings.TrimSpace(prompt)

			llm, err := anthropic.NewClient(&config.ModelConfig{
				Type:      "anthropic",
				Model:     "claude-sonnet-4-0",
				MaxTokens: 64000,
			}, environment.NewEnvVariableProvider(), logger)
			if err != nil {
				return fmt.Errorf("failed to create LLM client: %w", err)
			}

			fmt.Println("Generating agent configuration....")

			agents := team.New(map[string]*agent.Agent{
				"root": agent.New("root",
					agentBuilderInstructions,
					agent.WithModel(llm),
					agent.WithToolSets([]tools.ToolSet{builtin.NewShellTool(), builtin.NewFilesystemTool([]string{"."})}),
				),
			})

			sess := session.New(logger, session.WithUserMessage(prompt))

			rt, err := runtime.New(logger, agents, "root")
			if err != nil {
				logger.Error("failed to create runtime", "error", err)
				return err
			}

			messages, err := rt.Run(ctx, sess)
			if err != nil {
				logger.Error("failed to run session", "error", err)
				return err
			}

			fmt.Println(messages[len(messages)-1].Message.Content)

			return nil
		},
	}

	return cmd
}
