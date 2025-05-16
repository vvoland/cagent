package root

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
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
	logger.Debug("Starting agent", "agent", agentName)

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}

	agents, err := config.Agents(configFile)
	if err != nil {
		return err
	}

	rt, err := runtime.New(cfg, logger, agents, agentName)
	if err != nil {
		return err
	}

	sess := session.New(cfg)

	if len(args) > 0 {
		sess.Messages = append(sess.Messages, session.AgentMessage{
			Agent: agents[agentName],
			Message: chat.ChatCompletionMessage{
				Role:    "user",
				Content: args[0],
			},
		})

		response, err := rt.Run(ctx, sess)
		if err != nil {
			return err
		}

		fmt.Println(response[len(response)-1].Message.Content)
		return nil
	}

	scanner := bufio.NewScanner(os.Stdin)

	blue := color.New(color.FgBlue).SprintfFunc()
	fmt.Println(blue("\nEnter your messages (Ctrl+C to exit):"))

	for {
		fmt.Print(blue("> "))

		if !scanner.Scan() {
			break
		}

		userInput := scanner.Text()

		if strings.TrimSpace(userInput) == "" {
			continue
		}

		sess.Messages = append(sess.Messages, session.AgentMessage{
			Agent: rt.CurrentAgent(),
			Message: chat.ChatCompletionMessage{
				Role:    "user",
				Content: userInput,
			},
		})

		first := false
		for event := range rt.RunStream(ctx, sess) {
			if !first {
				fmt.Printf("%s", blue("[%s]: ", rt.CurrentAgent().Name()))
				first = true
			}
			switch event.(type) {
			case *runtime.AgentChoiceEvent:
				fmt.Printf("%s", event.(*runtime.AgentChoiceEvent).Choice.Delta.Content)
			case *runtime.ToolCallEvent:
				fmt.Printf("%s(%s)\n", event.(*runtime.ToolCallEvent).ToolCall.Function.Name, event.(*runtime.ToolCallEvent).ToolCall.Function.Arguments)
			case *runtime.AgentMessageEvent:
				fmt.Printf("%s\n", event.(*runtime.AgentMessageEvent).Message.Content)
			case *runtime.ErrorEvent:
				fmt.Printf("%s\n", event.(*runtime.ErrorEvent).Error)
			}
		}
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
