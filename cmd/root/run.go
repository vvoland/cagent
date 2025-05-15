package root

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/rumpl/cagent/config"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
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

	agents, err := config.Agents(configFile)
	if err != nil {
		return err
	}

	runtime, err := runtime.NewRuntime(cfg, logger, agents, agentName)
	if err != nil {
		return err
	}

	sess := session.New(cfg)

	if len(args) > 0 {
		sess.Messages = append(sess.Messages, session.AgentMessage{
			Agent: agents[agentName],
			Message: openai.ChatCompletionMessage{
				Role:    "user",
				Content: args[0],
			},
		})

		response, err := runtime.Run(ctx, sess)
		if err != nil {
			return err
		}

		fmt.Println(response[len(response)-1].Message.Content)
		return nil
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\nEnter your messages (Ctrl+C to exit):")

	for {
		fmt.Print("> ")

		if !scanner.Scan() {
			break
		}

		userInput := scanner.Text()

		if strings.TrimSpace(userInput) == "" {
			continue
		}

		sess.Messages = append(sess.Messages, session.AgentMessage{
			Agent: runtime.CurrentAgent(),
			Message: openai.ChatCompletionMessage{
				Role:    "user",
				Content: userInput,
			},
		})

		response, err := runtime.Run(ctx, sess)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("[%s]: %s\n", runtime.CurrentAgent().GetName(), response[len(response)-1].Message.Content)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
