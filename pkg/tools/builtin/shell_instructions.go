package builtin

// nativeInstructions contains the usage guide for native shell mode.
const nativeInstructions = `# Shell Tool Usage Guide

Execute shell commands in the user's environment with full control over working directories and command parameters.

## Core Concepts

**Execution Context**: Commands run in the user's default shell with access to all environment variables and the current workspace.
On Windows, PowerShell (pwsh/powershell) is used when available; otherwise, cmd.exe is used.
On Unix-like systems, ${SHELL} is used or /bin/sh as fallback.

**Working Directory Management**:
- Default execution location: working directory of the agent
- Override with "cwd" parameter for targeted command execution
- Supports both absolute and relative paths

**Command Isolation**: Each tool call creates a fresh shell session - no state persists between executions.

**Timeout Protection**: Commands have a default 30-second timeout to prevent hanging. For longer operations, specify a custom timeout.

## Parameter Reference

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| cmd       | string | Yes      | Shell command to execute |
| cwd       | string | Yes      | Working directory (use "." for current) |
| timeout   | int    | No       | Timeout in seconds (default: 30) |

## Best Practices

### ✅ DO
- Leverage the "cwd" parameter for directory-specific commands, rather than cding within commands
- Quote arguments containing spaces or special characters
- Use pipes and redirections
- Write advanced scripts with heredocs, that replace a lot of simple commands or tool calls
- This tool is great at reading and writing multiple files at once
- Avoid writing shell scripts to the disk. Instead, use heredocs to pipe the script to the SHELL
- Use the timeout parameter for long-running operations (e.g., builds, tests)

### Git Commits

When user asks to create git commit

- Add "Assisted-By: cagent" as a trailer line in the commit message
- Use the format: git commit -m "Your commit message" -m "" -m "Assisted-By: cagent"

## Usage Examples

**Basic command execution:**
{ "cmd": "ls -la", "cwd": "." }

**Long-running command with custom timeout:**
{ "cmd": "npm run build", "cwd": ".", "timeout": 120 }

**Language-specific operations:**
{ "cmd": "go test ./...", "cwd": ".", "timeout": 180 }
{ "cmd": "npm install", "cwd": "frontend" }
{ "cmd": "python -m pytest tests/", "cwd": "backend", "timeout": 90 }

**File operations:**
{ "cmd": "find . -name '*.go' -type f", "cwd": "." }
{ "cmd": "grep -r 'TODO' src/", "cwd": "." }

**Process management:**
{ "cmd": "ps aux | grep node", "cwd": "." }
{ "cmd": "docker ps --format 'table {{.Names}}\t{{.Status}}'", "cwd": "." }

**Complex pipelines:**
{ "cmd": "cat package.json | jq '.dependencies'", "cwd": "frontend" }

**Bash scripts:**
{ "cmd": "cat << 'EOF' | ${SHELL}
echo Hello
EOF" }

## Error Handling

Commands that exit with non-zero status codes will return error information along with any output produced before failure.
Commands that exceed their timeout will be terminated automatically.

---

# Background Jobs

Run long-running processes in the background while continuing with other tasks. Perfect for starting servers, watching files, or any process that needs to run alongside other operations.

## When to Use Background Jobs

**Use background jobs for:**
- Starting web servers, databases, or other services
- Running file watchers or live reload tools
- Long-running processes that other tasks depend on
- Commands that produce continuous output over time

**Don't use background jobs for:**
- Quick commands that complete in seconds
- Commands where you need immediate results
- One-time operations (use regular shell tool instead)

## Background Job Tools

**run_background_job**: Start a command in the background
- Parameters: cmd (required), cwd (optional, defaults to ".")
- Returns: Job ID for tracking

**list_background_jobs**: List all background jobs
- No parameters required
- Returns: Status, runtime, and command for each job

**view_background_job**: View output of a specific job
- Parameters: job_id (required)
- Returns: Current output and job status

**stop_background_job**: Stop a running job
- Parameters: job_id (required)
- Terminates the process and all child processes

## Background Job Workflow

**1. Start a background job:**
{ "cmd": "npm start", "cwd": "frontend" }
→ Returns job ID (e.g., "job_1731772800_1")

**2. Check running jobs:**
Use list_background_jobs to see all jobs with their status

**3. View job output:**
{ "job_id": "job_1731772800_1" }
→ Shows current output and status

**4. Stop job when done:**
{ "job_id": "job_1731772800_1" }
→ Terminates the process and all child processes

## Important Characteristics

**Output Buffering**: Each job captures up to 10MB of output. Beyond this limit, new output is discarded to prevent memory issues.

**Process Groups**: Background jobs and all their child processes are managed as a group. Stopping a job terminates the entire process tree.

**Environment Inheritance**: Jobs inherit environment variables from when they are started. Changes after job start don't affect running jobs.

**Automatic Cleanup**: All background jobs are automatically terminated when the agent stops.

**Job Persistence**: Job history is kept in memory until agent stops. Completed jobs remain queryable.

## Background Job Examples

**Start a web server:**
{ "cmd": "python -m http.server 8000", "cwd": "." }

**Start a development server:**
{ "cmd": "npm run dev", "cwd": "frontend" }

**Run a file watcher:**
{ "cmd": "go run . watch", "cwd": "." }

**Start a database:**
{ "cmd": "docker run --rm -p 5432:5432 postgres:latest", "cwd": "." }

**Multiple services pattern:**
1. Start backend: run_background_job with server command
2. Start frontend: run_background_job with dev server
3. Perform tasks: use other tools while services run
4. Check logs: view_background_job to see service output
5. Cleanup: stop_background_job for each service (or let agent cleanup automatically)`
