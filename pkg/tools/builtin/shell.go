package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/docker/cagent/pkg/tools"
)

type ShellTool struct {
	handler *shellHandler
}

// Make sure Shell Tool implements the ToolSet Interface
var _ tools.ToolSet = (*ShellTool)(nil)

type shellHandler struct {
	shell string
}

func (h *shellHandler) CallTool(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var params struct {
		Cmd string `json:"cmd"`
		Cwd string `json:"cwd"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	cmd := exec.CommandContext(ctx, h.shell, "-c", params.Cmd)
	cmd.Env = os.Environ()
	if params.Cwd != "" {
		cmd.Dir = params.Cwd
	} else {
		cmd.Dir = os.Getenv("PWD")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &tools.ToolCallResult{
			Output: fmt.Sprintf("Error executing command: %s\nOutput: %s", err, string(output)),
		}, nil
	}

	return &tools.ToolCallResult{
		Output: fmt.Sprintf("Output: %s", string(output)),
	}, nil
}

func NewShellTool() *ShellTool {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh" // Fallback to /bin/sh if SHELL is not set
	}

	return &ShellTool{
		handler: &shellHandler{
			shell: shell,
		},
	}
}

func (t *ShellTool) Instructions() string {
	return `# Shell Tool Usage Guide

Execute shell commands in the user's environment with full control over working directories and command parameters.

## Core Concepts

**Execution Context**: Commands run in the user's default shell (${SHELL}) with access to all environment variables and the current workspace.

**Working Directory Management**:
- Default execution location: workspace root
- Override with "cwd" parameter for targeted command execution
- Supports both absolute and relative paths

**Command Isolation**: Each tool call creates a fresh shell session - no state persists between executions.

## Parameter Reference

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| cmd       | string | Yes      | Shell command to execute |
| cwd       | string | Yes      | Working directory (use "." for current) |

## Best Practices

### ✅ DO
- Use separate tool calls for independent operations
- Leverage the "cwd" parameter for directory-specific commands
- Quote arguments containing spaces or special characters
- Use pipes and redirections within a single command

### ❌ AVOID
- Chaining unrelated commands with ";" or "&&"
- Relying on state from previous commands
- Complex multi-line scripts (break into separate calls)

## Usage Examples

**Basic command execution:**
{ "cmd": "ls -la", "cwd": "." }

**Language-specific operations:**
{ "cmd": "go test ./...", "cwd": "." }
{ "cmd": "npm install", "cwd": "frontend" }
{ "cmd": "python -m pytest tests/", "cwd": "backend" }

**File operations:**
{ "cmd": "find . -name '*.go' -type f", "cwd": "." }
{ "cmd": "grep -r 'TODO' src/", "cwd": "." }

**Process management:**
{ "cmd": "ps aux | grep node", "cwd": "." }
{ "cmd": "docker ps --format 'table {{.Names}}\t{{.Status}}'", "cwd": "." }

**Complex pipelines:**
{ "cmd": "cat package.json | jq '.dependencies'", "cwd": "frontend" }

## Error Handling

Commands that exit with non-zero status codes will return error information along with any output produced before failure.`
}

func (t *ShellTool) Tools(context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Function: &tools.FunctionDefinition{
				Name:        "shell",
				Description: `Executes the given shell command in the user's default shell.`,
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"cmd": map[string]any{
							"type":        "string",
							"description": "The shell command to execute",
						},
						"cwd": map[string]any{
							"type":        "string",
							"description": "The working directory to execute the command in",
						},
					},
					Required: []string{"cmd", "cwd"},
				},
			},
			Handler: t.handler.CallTool,
		},
	}, nil
}

func (t *ShellTool) Start(context.Context) error {
	return nil
}

func (t *ShellTool) Stop() error {
	return nil
}
