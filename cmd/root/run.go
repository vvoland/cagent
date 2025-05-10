package root

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rumpl/cagent/agent"
	"github.com/rumpl/cagent/config"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

// NewRunCmd creates a new run command
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run an agent",
		Long:  `Run an agent with the specified configuration and prompt`,
		RunE:  runAgentCommand,
	}
	// Add flags
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "agent.yaml", "Path to the configuration file")
	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")

	return cmd
}

func runAgentCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.Default()
	logger.Info("Starting agent", "agent", agentName)

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}

	rootAgent, err := agent.New(cfg, agentName)
	if err != nil {
		return err
	}

	runtime, err := runtime.NewRuntime(cfg, logger)
	if err != nil {
		return err
	}

	response, err := runtime.Run(ctx, rootAgent, []openai.ChatCompletionMessage{{Role: "user", Content: args[0]}})
	if err != nil {
		return err
	}

	fmt.Println(response[len(response)-1].Content)

	return nil
}
