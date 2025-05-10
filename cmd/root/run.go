package root

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/rumpl/cagent/agent"
	"github.com/rumpl/cagent/config"
	"github.com/rumpl/cagent/pkg/runtime"
	goOpenAI "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

// NewRunCmd creates a new run command
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run an agent",
		Long:  `Run an agent with the specified configuration and prompt`,
		Run:   runAgentCommand,
	}
	// Add flags
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "agent.yaml", "Path to the configuration file")
	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")

	return cmd
}

func runAgentCommand(cmd *cobra.Command, args []string) {
	prompt := args[0]
	logger := slog.Default()
	logger.Info("Starting agent", "agent", agentName)

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a context that can be canceled
	ctx := context.Background()

	// Create the agent
	a, err := agent.New(cfg, agentName)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create a runtime for the agent
	runtime, err := runtime.NewRuntime(cfg)
	if err != nil {
		log.Fatalf("Failed to create runtime: %v", err)
	}

	messages := []goOpenAI.ChatCompletionMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Run the agent runtime
	response, err := runtime.Run(ctx, a, messages)
	if err != nil {
		log.Fatalf("Agent error: %v", err)
	}
	fmt.Println(response[len(response)-1].Content)
}
