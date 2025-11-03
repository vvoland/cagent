package root

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/teamloader"
	"github.com/docker/cagent/pkg/telemetry"
	"github.com/docker/cagent/pkg/version"
)

type mcpFlags struct {
	runConfig config.RuntimeConfig
}

func newMCPCmd() *cobra.Command {
	var flags mcpFlags

	cmd := &cobra.Command{
		Use:   "mcp <agent-file>",
		Short: "Start an MCP (Model Context Protocol) server",
		Long:  `Start an MCP server that exposes agents as MCP tools via stdio`,
		Example: `  cagent mcp ./agent.yaml
  cagent mcp ./team.yaml
  cagent mcp agentcatalog/pirate`,
		Args: cobra.ExactArgs(1),
		RunE: flags.runMCPCommand,
	}

	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *mcpFlags) runMCPCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("mcp", args)
	return f.runMCP(cmd, args)
}

func (f *mcpFlags) runMCP(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	slog.Debug("Starting MCP server", "agent_ref", args[0])

	agentFilename, err := agentfile.Resolve(ctx, args[0])
	if err != nil {
		return err
	}

	if f.runConfig.RedirectURI == "" {
		f.runConfig.RedirectURI = "http://localhost:8083/oauth-callback"
	}

	t, err := teamloader.Load(ctx, agentFilename, f.runConfig)
	if err != nil {
		return fmt.Errorf("failed to load agents: %w", err)
	}

	defer func() {
		if err := t.StopToolSets(ctx); err != nil {
			slog.Error("Failed to stop tool sets", "error", err)
		}
	}()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "cagent",
		Version: version.Version,
	}, nil)

	agentNames := t.AgentNames()
	slog.Debug("Adding MCP tools for agents", "count", len(agentNames))

	for _, agentName := range agentNames {
		agent, err := t.Agent(agentName)
		if err != nil {
			return fmt.Errorf("failed to get agent %s: %w", agentName, err)
		}

		description := agent.Description()
		if description == "" {
			description = fmt.Sprintf("Run the %s agent", agentName)
		}

		slog.Debug("Adding MCP tool", "agent", agentName, "description", description)

		toolDef := &mcp.Tool{
			Name:        agentName,
			Description: description,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "The message to send to the agent",
					},
				},
				"required": []string{"message"},
			},
		}

		mcp.AddTool(server, toolDef, createToolHandler(t, agentName, agentFilename))
	}

	slog.Debug("MCP server starting with stdio transport")

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}

type ToolInput struct {
	Message string `json:"message" jsonschema:"the message to send to the agent"`
}

type ToolOutput struct {
	Response string `json:"response" jsonschema:"the response from the agent"`
}

func createToolHandler(t *team.Team, agentName, agentFilename string) func(context.Context, *mcp.CallToolRequest, ToolInput) (*mcp.CallToolResult, ToolOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ToolInput) (*mcp.CallToolResult, ToolOutput, error) {
		slog.Debug("MCP tool called", "agent", agentName, "message", input.Message)

		agent, err := t.Agent(agentName)
		if err != nil {
			return nil, ToolOutput{}, fmt.Errorf("failed to get agent: %w", err)
		}

		sess := session.New(
			session.WithTitle("MCP tool call"),
			session.WithMaxIterations(agent.MaxIterations()),
			session.WithUserMessage(agentFilename, input.Message),
		)
		sess.ToolsApproved = true

		rt, err := runtime.New(t,
			runtime.WithCurrentAgent(agentName),
			runtime.WithRootSessionID(sess.ID),
		)
		if err != nil {
			return nil, ToolOutput{}, fmt.Errorf("failed to create runtime: %w", err)
		}

		_, err = rt.Run(ctx, sess)
		if err != nil {
			slog.Error("Agent execution failed", "agent", agentName, "error", err)
			return nil, ToolOutput{}, fmt.Errorf("agent execution failed: %w", err)
		}

		result := sess.GetLastAssistantMessageContent()
		if result == "" {
			result = "No response from agent"
		}

		slog.Debug("Agent execution completed", "agent", agentName, "response_length", len(result))

		return nil, ToolOutput{Response: result}, nil
	}
}
