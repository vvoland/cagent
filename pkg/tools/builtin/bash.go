package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/rumpl/cagent/pkg/tools"
)

type BashTool struct {
	handler *bashHandler
}

type bashHandler struct {
	shell string
}

func (h *bashHandler) CallTool(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
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

func NewBashTool() *BashTool {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh" // Fallback to /bin/sh if SHELL is not set
	}

	return &BashTool{
		handler: &bashHandler{
			shell: shell,
		},
	}
}

func (t *BashTool) Instructions() string {
	return `## Important notes about the "bash" tool

1. Directory Verification:
   - If the command will create new directories or files, first use the list_directory tool to verify the parent directory exists and is the correct location
   - For example, before running a mkdir command, first use list_directory to check the parent directory exists

2. Working directory:
   - Your working directory is at the root of the user's workspace unless you override it for a command by setting the "cwd" parameter.
   - Use the "cwd" parameter to specify an absolute or relative path to a directory where the command should be executed (e.g., "cwd: "core/src"").
   - You may use "cd PATH && COMMAND" if the user explicitly requests it, otherwise prefer using the "cwd" parameter.

3. Multiple independent commands:
   - Do NOT chain multiple independent commands with ";".
   - Instead, make multiple separate tool calls for each command you want to run

4. Shell escapes:
   - Escape any special characters in the command if those are not to be interpreted by the shell

5. Truncated output:
   - Only the last 50000 characters of the output will be returned to you along with how many lines got truncated, if any
   - If necessary, when the output is truncated, consider running the command again with a grep or head filter to search through the truncated lines

6. Stateless environment:
   - Setting an environment variable or using "cd" only impacts a single command, it does not persist between commands

## Examples

- To run 'go test ./...': use { cmd: 'go test ./...' }
- To run 'cargo build' in the core/src directory: use { cmd: 'cargo build', cwd: 'core/src' }
- To run 'ps aux | grep node', use { cmd: 'ps aux | grep node' }
- To run commands in a subdirectory using cd: use { cmd: 'cd core/src && ls -la' }
- To print a special character like $ with some command "cmd", use { cmd: 'cmd \$' }

## Git

Use this tool to interact with git. You can use it to run 'git log', 'git show', or other 'git' commands.

When the user shares a git commit SHA, you can use 'git show' to look it up. When the user asks when a change was introduced, you can use 'git log'.

If the user asks you to, use this tool to create git commits too. But only if the user asked.

<git-example>
user: commit the changes
assistant: [uses Bash to run 'git status']
[uses Bash to 'git add' the changes from the 'git status' output]
[uses Bash to run 'git commit -m "commit message"']
</git-example>

<git-example>
user: commit the changes
assistant: [uses Bash to run 'git status']
there are already files staged, do you want me to add the changes?
user: yes
assistant: [uses Bash to 'git add' the unstaged changes from the 'git status' output]
[uses Bash to run 'git commit -m "commit message"']
</git-example>

## Prefer specific tools

It's VERY IMPORTANT to use specific tools when searching for files, instead of issuing terminal commands with find/grep/ripgrep. Use codebase_search or Grep instead. Use read_file tool rather than cat, and edit_file rather than sed.`
}

func (t *BashTool) Tools(ctx context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Function: &tools.FunctionDefinition{
				Name:        "bash",
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

func (t *BashTool) Start(ctx context.Context) error {
	return nil
}

func (t *BashTool) Stop() error {
	return nil
}
