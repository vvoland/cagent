---
title: "CLI Reference"
description: "Complete reference for all cagent command-line commands and flags."
permalink: /features/cli/
---

# CLI Reference

_Complete reference for all cagent command-line commands and flags._

<div class="callout callout-tip">
<div class="callout-title">💡 No config needed
</div>
  <p>Running <code>cagent run</code> without a config file uses a built-in default agent. Perfect for quick experimentation.</p>

</div>

## Commands

### `cagent run`

Launch the interactive TUI with an agent configuration.

```bash
cagent run [config] [message...] [flags]
```

| Flag                         | Description                                                                                                                               |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `-a, --agent &lt;name&gt;`   | Run a specific agent from the config                                                                                                      |
| `--yolo`                     | Auto-approve all tool calls                                                                                                               |
| `--model &lt;ref&gt;`        | Override model(s). Use `provider/model` for all agents, or `agent=provider/model` for specific agents. Comma-separate multiple overrides. |
| `--session &lt;id&gt;`       | Resume a previous session. Supports relative refs (`-1` = last, `-2` = second to last)                                                    |
| `--prompt-file &lt;path&gt;` | Include file contents as additional system context (repeatable)                                                                           |
| `-c &lt;name&gt;`            | Run a named command from the YAML config                                                                                                  |
| `-d, --debug`                | Enable debug logging                                                                                                                      |
| `--log-file &lt;path&gt;`    | Custom debug log location                                                                                                                 |
| `-o, --otel`                 | Enable OpenTelemetry tracing                                                                                                              |

```bash
# Examples
$ cagent run agent.yaml
$ cagent run agent.yaml "Fix the bug in auth.go"
$ cagent run agent.yaml -a developer --yolo
$ cagent run agent.yaml --model anthropic/claude-sonnet-4-0
$ cagent run agent.yaml --model "dev=openai/gpt-4o,reviewer=anthropic/claude-sonnet-4-0"
$ cagent run agent.yaml --session -1  # resume last session
$ cagent run agent.yaml -c df         # run named command
$ cagent run agent.yaml --prompt-file ./context.md  # include file as context

# Queue multiple messages (processed in sequence)
$ cagent run agent.yaml "question 1" "question 2" "question 3"
```

### `cagent run --exec`

Run an agent in non-interactive (headless) mode. No TUI — output goes to stdout.

```bash
cagent run --exec [config] [message...] [flags]
```

```bash
# One-shot task
$ cagent run --exec agent.yaml "Create a Dockerfile for a Python Flask app"

# With auto-approve
$ cagent run --exec agent.yaml --yolo "Set up CI/CD pipeline"

# Multi-turn conversation
$ cagent run --exec agent.yaml "question 1" "question 2" "question 3"
```

### `cagent new`

Interactively generate a new agent configuration file.

```bash
$ cagent new [flags]

# Examples
$ cagent new
$ cagent new --model openai/gpt-5-mini --max-tokens 32000
$ cagent new --model dmr/ai/gemma3-qat:12B --max-iterations 15
```

### `cagent api`

Start the HTTP API server for programmatic access.

```bash
$ cagent api [config] [flags]

# Examples
$ cagent api agent.yaml
$ cagent api agent.yaml --listen :8080
$ cagent api ociReference --pull-interval 10  # auto-refresh
```

### `cagent mcp`

Expose agents as MCP tools for use in Claude Desktop, Claude Code, or other MCP clients.

```bash
$ cagent mcp [config] [flags]

# Examples
$ cagent mcp agent.yaml
$ cagent mcp agent.yaml --working-dir /path/to/project
$ cagent mcp agentcatalog/coder
```

See [MCP Mode](/features/mcp-mode/) for detailed setup.

### `cagent serve a2a`

Start an A2A (Agent-to-Agent) protocol server.

```bash
$ cagent serve a2a [config] [flags]

# Examples
$ cagent serve a2a agent.yaml
$ cagent serve a2a agent.yaml --listen 127.0.0.1:9000
```

### `cagent acp`

Start an ACP (Agent Client Protocol) server over stdio. This allows external clients to interact with your agents using the ACP protocol.

```bash
$ cagent acp [config] [flags]

# Examples
$ cagent acp agent.yaml
```

See [ACP](/features/acp/) for details on the Agent Client Protocol.

### `cagent share push` / `cagent pull`

Share agents via OCI registries.

```bash
# Push an agent
$ cagent share push ./agent.yaml docker.io/username/my-agent:latest

# Pull an agent
$ cagent share pull docker.io/username/my-agent:latest
```

See [Agent Distribution](/concepts/distribution/) for full registry workflow details.

### `cagent eval`

Run agent evaluations.

```bash
$ cagent eval eval-config.yaml

# With flags
$ cagent eval agent.yaml ./evals -c 8              # 8 concurrent evaluations
$ cagent eval agent.yaml --keep-containers         # Keep containers for debugging
$ cagent eval agent.yaml --only "auth*"            # Only run matching evals
```

### `cagent alias`

Manage agent aliases for quick access.

```bash
# List aliases
$ cagent alias ls

# Add an alias
$ cagent alias add pirate /path/to/pirate.yaml
$ cagent alias add other ociReference

# Add an alias with runtime options
$ cagent alias add yolo-coder agentcatalog/coder --yolo
$ cagent alias add fast-coder agentcatalog/coder --model openai/gpt-4o-mini
$ cagent alias add turbo agentcatalog/coder --yolo --model anthropic/claude-sonnet-4-0

# Use an alias
$ cagent run pirate
$ cagent run yolo-coder
```

**Alias Options:** Aliases can include runtime options that apply automatically when used:

- `--yolo` — Auto-approve all tool calls when running the alias
- `--model &lt;ref&gt;` — Override the model for the alias

When listing aliases, options are shown in brackets:

```bash
$ cagent alias ls
Registered aliases (3):

  fast-coder  → agentcatalog/coder [model=openai/gpt-4o-mini]
  turbo       → agentcatalog/coder [yolo, model=anthropic/claude-sonnet-4-0]
  yolo-coder  → agentcatalog/coder [yolo]

Run an alias with: cagent run <alias>
```

<div class="callout callout-tip">
<div class="callout-title">💡 Override alias options
</div>
  <p>Command-line flags override alias options. For example, <code>cagent run yolo-coder --yolo=false</code> disables yolo mode even though the alias has it enabled.</p>

</div>

## Global Flags

| Flag                      | Description                                                  |
| ------------------------- | ------------------------------------------------------------ |
| `-d, --debug`             | Enable debug logging (default: `~/.cagent/cagent.debug.log`) |
| `--log-file &lt;path&gt;` | Custom debug log location                                    |
| `-o, --otel`              | Enable OpenTelemetry tracing                                 |
| `--help`                  | Show help for any command                                    |

## Agent References

Commands that accept a config support multiple reference types:

| Type          | Example                                     |
| ------------- | ------------------------------------------- |
| Local file    | `./agent.yaml`                              |
| OCI registry  | `docker.io/username/agent:latest`           |
| Agent catalog | `agentcatalog/pirate`                       |
| Alias         | `pirate` (after `cagent alias add`)         |
| Default       | (no argument) — uses built-in default agent |

<div class="callout callout-info">
<div class="callout-title">ℹ️ Debugging
</div>
  <p>Having issues? See <a href="/community/troubleshooting/">Troubleshooting</a> for debug mode, log analysis, and common solutions.</p>

</div>
