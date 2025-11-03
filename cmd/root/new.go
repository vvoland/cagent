package root

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/creator"
	"github.com/docker/cagent/pkg/input"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/telemetry"
)

type newFlags struct {
	modelParam         string
	maxTokensParam     int
	maxIterationsParam int
	runConfig          config.RuntimeConfig
}

func newNewCmd() *cobra.Command {
	var flags newFlags

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new agent configuration",
		Long:  `Create a new agent configuration by asking questions and generating a YAML file`,
		RunE:  flags.runNewCommand,
	}

	cmd.PersistentFlags().StringVar(&flags.modelParam, "model", "", "Model to use, optionally as provider/model where provider is one of: anthropic, openai, google, dmr. If omitted, provider is auto-selected based on available credentials or gateway")
	cmd.PersistentFlags().IntVar(&flags.maxTokensParam, "max-tokens", 0, "Override max_tokens for the selected model (0 = default)")
	cmd.PersistentFlags().IntVar(&flags.maxIterationsParam, "max-iterations", 0, "Maximum number of agentic loop iterations to prevent infinite loops (default: 20 for DMR, unlimited for other providers)")

	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *newFlags) runNewCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("new", args)

	ctx := cmd.Context()
	out := cli.NewPrinter(cmd.OutOrStdout())

	var model string         // final model name
	var modelProvider string // final provider name

	// Parse provider from --model if specified as "provider/model" where provider is recognized
	derivedProvider := ""
	if idx := strings.Index(f.modelParam, "/"); idx > 0 {
		candidate := strings.ToLower(f.modelParam[:idx])
		switch candidate {
		case "anthropic", "openai", "google", "dmr":
			derivedProvider = candidate
			model = f.modelParam[idx+1:]
		}
	}

	// Determine provider
	if derivedProvider != "" {
		modelProvider = derivedProvider
	} else {
		if f.runConfig.ModelsGateway == "" {
			// Prefer Anthropic, then OpenAI, then Google based on available API keys
			// default to DMR if no provider credentials are found
			switch {
			case os.Getenv("ANTHROPIC_API_KEY") != "":
				modelProvider = "anthropic"
				out.Printf("%s\n\n", "ANTHROPIC_API_KEY found, using Anthropic")
			case os.Getenv("OPENAI_API_KEY") != "":
				modelProvider = "openai"
				out.Printf("%s\n\n", "OPENAI_API_KEY found, using OpenAI")
			case os.Getenv("GOOGLE_API_KEY") != "":
				modelProvider = "google"
				out.Printf("%s\n\n", "GOOGLE_API_KEY found, using Google")
			default:
				modelProvider = "dmr"
				out.Printf("%s\n\n", "⚠️ No provider credentials found, defaulting to Docker Model Runner (DMR)")
			}
			if f.modelParam == "" {
				out.Printf("%s\n\n", "use \"--model provider/model\" to use a different model")
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
		out.Printf("------- Welcome to %s! -------\n", AppName)
		out.Printf("%s\n\n", "         (Ctrl+C to exit)")
		out.Printf("%s\n\n", "What should your agent/agent team do? (describe its purpose)")
		out.Print("> ")

		var err error
		prompt, err = input.ReadLine(ctx, os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read purpose: %w", err)
		}
		prompt = strings.TrimSpace(prompt)
		out.Println()
	}

	events, rt, err := creator.StreamCreateAgent(ctx, ".", prompt, f.runConfig, modelProvider, model, f.maxTokensParam, f.maxIterationsParam)
	if err != nil {
		return err
	}

	llmIsTyping := false

	for event := range events {
		switch e := event.(type) {
		case *runtime.AgentChoiceEvent:
			if !llmIsTyping {
				out.Println()
				llmIsTyping = true
			}
			out.Printf("%s", e.Content)
		case *runtime.ToolCallEvent:
			if llmIsTyping {
				out.Println()
				llmIsTyping = false
			}
			out.PrintToolCall(e.ToolCall)
		case *runtime.ToolCallResponseEvent:
			if llmIsTyping {
				out.Println()
				llmIsTyping = false
			}
			out.PrintToolCallResponse(e.ToolCall, e.Response)
		case *runtime.ErrorEvent:
			if llmIsTyping {
				out.Println()
				llmIsTyping = false
			}
			out.PrintError(fmt.Errorf("%s", e.Error))
		case *runtime.MaxIterationsReachedEvent:
			if llmIsTyping {
				out.Println()
				llmIsTyping = false
			}

			result := out.PromptMaxIterationsContinue(ctx, e.MaxIterations)
			switch result {
			case cli.ConfirmationApprove:
				rt.Resume(ctx, runtime.ResumeTypeApprove)
			case cli.ConfirmationReject:
				rt.Resume(ctx, runtime.ResumeTypeReject)
				return nil
			case cli.ConfirmationAbort:
				rt.Resume(ctx, runtime.ResumeTypeReject)
			}
		}
	}
	out.Println()
	out.Println()
	return nil
}
