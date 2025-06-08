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
	"github.com/rumpl/cagent/pkg/loader"
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
	cmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug logging")

	return cmd
}

func runAgentCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Configure logger based on debug flag
	logLevel := slog.LevelInfo
	if debugMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Debug("Starting agent", "agent", agentName, "debug_mode", debugMode)

	agents, err := loader.Agents(ctx, configFile, logger)
	if err != nil {
		return err
	}

	rt, err := runtime.New(logger, agents, agentName)
	if err != nil {
		return err
	}

	sess := session.New(logger)

	if len(args) > 0 {
		sess.Messages = append(sess.Messages, session.AgentMessage{
			Agent: agents.Get(agentName),
			Message: chat.Message{
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
	yellow := color.New(color.FgYellow).SprintfFunc()
	green := color.New(color.FgGreen).SprintfFunc()
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
			Message: chat.Message{
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
			switch e := event.(type) {
			case *runtime.AgentChoiceEvent:
				fmt.Printf("%s", e.Choice.Delta.Content)
			case *runtime.ToolCallEvent:
				fmt.Printf("%s", yellow("%s(%s)\n", e.ToolCall.Function.Name, e.ToolCall.Function.Arguments))
			case *runtime.ToolCallResponseEvent:
				fmt.Printf("%s", green("done(%s)\n", e.ToolCall.Function.Name))
			case *runtime.AgentMessageEvent:
				fmt.Printf("%s\n", e.Message.Content)
			case *runtime.ErrorEvent:
				fmt.Printf("%s\n", e.Error)
			}
		}
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
