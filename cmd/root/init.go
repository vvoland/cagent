package root

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/model/provider/anthropic"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInitCmd creates a new init command
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new agent configuration",
		Long:  `Initialize a new agent configuration by asking questions and generating a YAML file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelWarn, // Use warn level for init to avoid verbose output
			}))

			reader := bufio.NewReader(os.Stdin)

			fmt.Print("What should your agent do? (describe its purpose): ")
			purpose, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read purpose: %w", err)
			}
			purpose = strings.TrimSpace(purpose)

			name := "root"

			fmt.Print("Agent description in one sentence: ")
			description, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}
			description = strings.TrimSpace(description)

			fmt.Print("Enable todo tracking? (y/n) [optional]: ")
			todoInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read todo: %w", err)
			}
			todo := strings.ToLower(strings.TrimSpace(todoInput)) == "y"

			fmt.Print("Enable thinking? (y/n) [optional]: ")
			thinkInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read think: %w", err)
			}
			think := strings.ToLower(strings.TrimSpace(thinkInput)) == "y"

			fmt.Print("Enable date awareness? (y/n) [optional]: ")
			dateInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read date awareness: %w", err)
			}
			addDate := strings.ToLower(strings.TrimSpace(dateInput)) == "y"

			llm, err := anthropic.NewClient(&config.ModelConfig{
				Type:      "anthropic",
				Model:     "claude-sonnet-4-0",
				MaxTokens: 64000,
			}, logger)
			if err != nil {
				return fmt.Errorf("failed to create LLM client: %w", err)
			}

			prompt := fmt.Sprintf(`Given this purpose for an AI agent: %q\n\nName: %s\nDescription: %s\n\nPlease write a system instruction that will guide the agent's behavior, try to infer what the user wants the agent to do.`, purpose, name, description)

			fmt.Println("\nGenerating agent instruction...")

			stream, err := llm.CreateChatCompletionStream(context.Background(), []chat.Message{
				{
					Role:    chat.MessageRoleUser,
					Content: prompt,
				},
			}, nil)
			if err != nil {
				return fmt.Errorf("failed to create completion stream: %w", err)
			}
			defer stream.Close()

			var instructionBuilder strings.Builder
			for {
				response, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("error receiving from stream: %w", err)
				}

				for _, choice := range response.Choices {
					instructionBuilder.WriteString(choice.Delta.Content)
				}
			}

			instruction := strings.TrimSpace(instructionBuilder.String())

			agent := config.AgentConfig{
				Name:        name,
				Model:       "anthropic",
				Description: description,
				Instruction: instruction,
				Todo:        todo,
				Think:       think,
				AddDate:     addDate,
			}
			agents := map[string]config.AgentConfig{
				name: agent,
			}
			models := map[string]config.ModelConfig{
				"anthropic": {
					Type:      "anthropic",
					Model:     "claude-sonnet-4-0",
					MaxTokens: 64000,
				},
			}
			cfg := config.Config{
				Agents: agents,
				Models: models,
			}

			out, err := yaml.Marshal(&cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal YAML: %w", err)
			}

			if err := os.WriteFile("agent.yaml", out, 0o644); err != nil {
				return fmt.Errorf("failed to write configuration file: %w", err)
			}

			fmt.Println("\nAgent configuration has been generated and saved to agent.yaml")
			fmt.Println("You can now run your agent using: cagent run agent.yaml")

			return nil
		},
	}

	return cmd
}
