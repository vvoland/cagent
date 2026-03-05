---
title: "CLI Reference"
description: "Complete reference for all docker-agent command-line commands and flags."
permalink: /features/cli/
---

# CLI Reference

_Complete reference for all docker-agent command-line commands and flags._

<div class="callout callout-tip">
<div class="callout-title">💡 No config needed
</div>
  <p>Running <code>docker agent run</code> without a config file uses a built-in default agent. Perfect for quick experimentation.</p>

</div>

## Commands

### `docker agent run`

Launch the interactive TUI with an agent configuration.

```bash
$ docker agent run [config] [message...] [flags]
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
$ docker agent run agent.yaml
$ docker agent run agent.yaml "Fix the bug in auth.go"
$ docker agent run agent.yaml -a developer --yolo
$ docker agent run agent.yaml --model anthropic/claude-sonnet-4-0
$ docker agent run agent.yaml --model "dev=openai/gpt-4o,reviewer=anthropic/claude-sonnet-4-0"
$ docker agent run agent.yaml --session -1  # resume last session
$ docker agent run agent.yaml -c df         # run named command
$ docker agent run agent.yaml --prompt-file ./context.md  # include file as context

# Queue multiple messages (processed in sequence)
$ docker agent run agent.yaml "question 1" "question 2" "question 3"
```

### `docker agent run --exec`

Run an agent in non-interactive (headless) mode. No TUI — output goes to stdout.

```bash
$ docker agent run --exec [config] [message...] [flags]
```

```bash
# One-shot task
$ docker agent run --exec agent.yaml "Create a Dockerfile for a Python Flask app"

# With auto-approve
$ docker agent run --exec agent.yaml --yolo "Set up CI/CD pipeline"

# Multi-turn conversation
$ docker agent run --exec agent.yaml "question 1" "question 2" "question 3"
```

### `docker agent new`

Interactively generate a new agent configuration file.

```bash
$ docker agent new [flags]

# Examples
$ docker agent new
$ docker agent new --model openai/gpt-5-mini --max-tokens 32000
$ docker agent new --model dmr/ai/gemma3-qat:12B --max-iterations 15
```

### `docker agent serve api`

Start the HTTP API server for programmatic access.

```bash
$ docker agent api [config] [flags]

# Examples
$ docker agent api agent.yaml
$ docker agent api agent.yaml --listen :8080
$ docker agent api ociReference --pull-interval 10  # auto-refresh
```

### `docker agent serve mcp`

Expose agents as MCP tools for use in Claude Desktop, Claude Code, or other MCP clients.

```bash
$ docker agent serve mcp [config] [flags]

# Examples
$ docker agent serve mcp agent.yaml
$ docker agent serve mcp agent.yaml --working-dir /path/to/project
$ docker agent serve mcp agentcatalog/coder
```

See [MCP Mode](/features/mcp-mode/) for detailed setup.

### `docker agent serve a2a`

Start an A2A (Agent-to-Agent) protocol server.

```bash
$ docker agent serve a2a [config] [flags]

# Examples
$ docker agent serve a2a agent.yaml
$ docker agent serve a2a agent.yaml --listen 127.0.0.1:9000
```

### `docker agent serve acp`

Start an ACP (Agent Client Protocol) server over stdio. This allows external clients to interact with your agents using the ACP protocol.

```bash
$ docker agent serve acp [config] [flags]

# Examples
$ docker agent serve acp agent.yaml
```

See [ACP](/features/acp/) for details on the Agent Client Protocol.

### `docker agent share push` / `docker agent pull`

Share agents via OCI registries.

```bash
# Push an agent
$ docker agent share push ./agent.yaml docker.io/username/my-agent:latest

# Pull an agent
$ docker agent share pull docker.io/username/my-agent:latest
```

See [Agent Distribution](/concepts/distribution/) for full registry workflow details.

### `docker agent eval`

Run agent evaluations.

```bash
$ docker agent eval eval-config.yaml

# With flags
$ docker agent eval agent.yaml ./evals -c 8              # 8 concurrent evaluations
$ docker agent eval agent.yaml --keep-containers         # Keep containers for debugging
$ docker agent eval agent.yaml --only "auth*"            # Only run matching evals
```

### `docker agent alias`

Manage agent aliases for quick access.

```bash
# List aliases
$ docker agent alias ls

# Add an alias
$ docker agent alias add pirate /path/to/pirate.yaml
$ docker agent alias add other ociReference

# Add an alias with runtime options
$ docker agent alias add yolo-coder agentcatalog/coder --yolo
$ docker agent alias add fast-coder agentcatalog/coder --model openai/gpt-4o-mini
$ docker agent alias add turbo agentcatalog/coder --yolo --model anthropic/claude-sonnet-4-0

# Use an alias
$ docker agent run pirate
$ docker agent run yolo-coder
```

**Alias Options:** Aliases can include runtime options that apply automatically when used:

- `--yolo` — Auto-approve all tool calls when running the alias
- `--model &lt;ref&gt;` — Override the model for the alias

When listing aliases, options are shown in brackets:

```bash
$ docker agent alias ls
Registered aliases (3):

  fast-coder  → agentcatalog/coder [model=openai/gpt-4o-mini]
  turbo       → agentcatalog/coder [yolo, model=anthropic/claude-sonnet-4-0]
  yolo-coder  → agentcatalog/coder [yolo]

Run an alias with: docker agent run <alias>
```

<div class="callout callout-tip">
<div class="callout-title">💡 Override alias options
</div>
  <p>Command-line flags override alias options. For example, <code>docker agent run yolo-coder --yolo=false</code> disables yolo mode even though the alias has it enabled.</p>

</div>

<div class="callout callout-tip">
<div class="callout-title">💡 Set a default agent
</div>
  <p>Create a <code>default</code> alias to customize what <code>docker agent</code> starts with no arguments:</p>
  <pre><code>$ docker agent alias add default /my/default/agent.yaml</code></pre>
  <p>Then simply run <code>docker agent</code> — it will launch that agent automatically.</p>

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
| Alias         | `pirate` (after `docker agent alias add`)   |
| Default       | (no argument) — uses built-in default agent |

<div class="callout callout-info">
<div class="callout-title">ℹ️ Debugging
</div>
  <p>Having issues? See <a href="{{ '/community/troubleshooting/' | relative_url }}">Troubleshooting</a> for debug mode, log analysis, and common solutions.</p>

</div>
