package new

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/internal/creator"
)

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

			out, _, err := creator.CreateAgent(ctx, ".", logger, prompt)
			if err != nil {
				return err
			}

			fmt.Println(out)

			return nil
		},
	}

	return cmd
}
