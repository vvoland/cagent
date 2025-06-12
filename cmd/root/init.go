package root

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/model/provider"
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
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("What should your agent do? (describe its purpose): ")
			purpose, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read purpose: %w", err)
			}
			purpose = strings.TrimSpace(purpose)

			name := "root"

			fmt.Print("Agent description (2-3 sentences): ")
			description, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}
			description = strings.TrimSpace(description)

			fmt.Print("Toolsets (comma separated, e.g. bash,task) [optional]: ")
			toolsetsInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read toolsets: %w", err)
			}
			toolsetsInput = strings.TrimSpace(toolsetsInput)
			var toolsets []config.Toolset
			if toolsetsInput != "" {
				for _, t := range strings.Split(toolsetsInput, ",") {
					name := strings.TrimSpace(t)
					if name != "" {
						toolsets = append(toolsets, config.Toolset{Type: name})
					}
				}
			}

			fmt.Print("Enable memory? (y/n) [optional]: ")
			memoryInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read memory: %w", err)
			}
			memory := strings.ToLower(strings.TrimSpace(memoryInput)) == "y"

			fmt.Print("Enable todo tracking? (y/n) [optional]: ")
			todoInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read todo: %w", err)
			}
			todo := strings.ToLower(strings.TrimSpace(todoInput)) == "y"

			fmt.Print("Enable date awareness? (y/n) [optional]: ")
			dateInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read date awareness: %w", err)
			}
			addDate := strings.ToLower(strings.TrimSpace(dateInput)) == "y"

			llm, err := provider.NewFactory().NewProvider(&config.ModelConfig{
				Type:  "anthropic",
				Model: "claude-3-5-sonnet-latest",
			})
			if err != nil {
				return fmt.Errorf("failed to create LLM client: %w", err)
			}

			prompt := fmt.Sprintf(`Given this purpose for an AI agent: %q\n\nName: %s\nDescription: %s\n\nPlease write a system instruction (2-3 sentences) that will guide the agent's behavior.`, purpose, name, description)

			fmt.Println("\nGenerating agent instruction...")

			stream, err := llm.CreateChatCompletionStream(context.Background(), []chat.Message{
				{
					Role:    "user",
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
					fmt.Print(choice.Delta.Content)
				}
			}

			instruction := strings.TrimSpace(instructionBuilder.String())

			fmt.Println("\n\nValidating configuration...")

			agent := config.AgentConfig{
				Name:        name,
				Model:       "anthropic",
				Description: description,
				Instruction: instruction,
				Toolsets:    toolsets,
				Memory:      memory,
				Todo:        todo,
				AddDate:     addDate,
			}
			agents := map[string]config.AgentConfig{
				name: agent,
			}
			models := map[string]config.ModelConfig{
				"anthropic": {
					Type:  "anthropic",
					Model: "claude-3-sonnet-latest",
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
			fmt.Println("You can now run your agent using: cagent run -c agent.yaml")

			return nil
		},
	}

	return cmd
}
