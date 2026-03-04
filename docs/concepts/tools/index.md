---
title: "Tools"
description: "Tools give agents the ability to interact with the world — read files, run commands, search the web, query databases, and more."
permalink: /concepts/tools/
---

# Tools

_Tools give agents the ability to interact with the world — read files, run commands, search the web, query databases, and more._

## How Tools Work

When an agent needs to perform an action, it makes a **tool call**. The docker-agent runtime executes the tool and returns the result to the agent, which can then use it to continue its work.

1. Agent receives a user message
2. Agent decides it needs to use a tool (e.g., read a file)
3. docker-agent executes the tool and returns the result
4. Agent incorporates the result and responds

<div class="callout callout-info">
<div class="callout-title">ℹ️ Tool Confirmation
</div>
  <p>By default, docker-agent asks for user confirmation before executing tools that have side effects (shell commands, file writes). Use <code>--yolo</code> to auto-approve all tool calls.</p>

</div>

## Built-in Tools

docker-agent ships with several built-in tools that require no external dependencies. Each is enabled by adding its `type` to the agent's `toolsets` list.

### Filesystem

Gives agents the ability to read, write, list, search, and navigate files and directories. The agent receives tools such as `read_file`, `write_file`, `list_directory`, `search_files_content`, `directory_tree`, and more.

```yaml
toolsets:
  - type: filesystem
```

The filesystem tool respects the working directory and allows agents to explore codebases, edit config files, create new files, and perform search-and-replace operations.

### Shell

Allows agents to execute arbitrary shell commands in the user's environment. This is one of the most powerful tools — it lets agents run builds, install dependencies, query APIs, and interact with the system.

```yaml
toolsets:
  - type: shell
```

Commands run in a fresh shell session with access to all environment variables. Each invocation is isolated — no state persists between calls. Requires user confirmation by default.

### Think

A reasoning scratchpad that lets agents think step-by-step before acting. The agent can write its thoughts without producing visible output to the user — useful for planning complex tasks, breaking down problems, and reasoning through multi-step solutions.

```yaml
toolsets:
  - type: think
```

This is a lightweight tool with no side effects. It's recommended for all agents — it improves the quality of reasoning on complex tasks at minimal cost.

### Todo

Task list management. Agents can create, update, list, and track progress on tasks. Useful for complex multi-step workflows where the agent needs to stay organized.

```yaml
toolsets:
  - type: todo
```

The agent gets tools like `create_todo`, `update_todos`, and `list_todos` with status tracking (pending, in-progress, completed).

### Memory

Persistent key-value storage backed by SQLite. Data survives across sessions, allowing agents to remember facts, user preferences, project context, and past decisions.

```yaml
toolsets:
  - type: memory
    path: ./agent_memory.db # optional: custom database path
```

Without `path`, a default location is used. Memory is especially useful for long-running assistants that need to recall information across conversations.

### Fetch

Make HTTP requests (GET, POST, PUT, DELETE, etc.) to external APIs. The agent can read web pages, call REST APIs, download data, and interact with web services.

```yaml
toolsets:
  - type: fetch
```

### Script

Define custom shell scripts as named tools. Unlike the generic `shell` tool where the agent writes the command, script tools execute predefined commands — useful for exposing safe, constrained operations.

```yaml
toolsets:
  - type: script
    scripts:
      - name: run_tests
        description: Run the project test suite
        command: task test
      - name: lint
        description: Run the linter
        command: task lint
```

### Transfer Task

The `transfer_task` tool is automatically available when an agent has `sub_agents` configured. It allows the agent to delegate tasks to specialized sub-agents and receive their results. This is the core mechanism for multi-agent orchestration — you don't need to add it manually.

## MCP Tools

docker-agent supports the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) for extending agents with external tools. There are three ways to connect MCP tools:

### Docker MCP (Recommended)

Run MCP servers in Docker containers for security and isolation:

```yaml
toolsets:
  - type: mcp
    ref: docker:duckduckgo # web search
  - type: mcp
    ref: docker:github-official # GitHub integration
```

Docker MCP tools run through the [Docker MCP Gateway](https://github.com/docker/mcp-gateway), which manages container lifecycles and provides security isolation. Browse available tools in the [Docker MCP Catalog](https://hub.docker.com/search?q=&type=mcp).

### Local MCP (stdio)

Run MCP servers as local processes communicating over stdin/stdout. Here's an example adding `rust-mcp-filesystem` for file operations alongside a Docker MCP tool:

```yaml
toolsets:
  - type: mcp
    ref: docker:duckduckgo
  - type: mcp
    command: rust-mcp-filesystem
    args: ["--allow-write", "."]
    tools: ["read_file", "write_file"] # optional: only expose specific tools
    env:
      - "RUST_LOG=debug"
```

### Remote MCP (SSE / HTTP)

Connect to MCP servers running on a network:

```yaml
toolsets:
  - type: mcp
    remote:
      url: "https://mcp.example.com"
      transport_type: "sse"
      headers:
        Authorization: "Bearer your-token"
```

## Tool Filtering

Toolsets may expose many tools. You can use the `tools` property to whitelist only the ones you need. This works for any toolset type — not just MCP:

```yaml
toolsets:
  - type: mcp
    ref: docker:github-official
    tools: ["list_issues", "create_issue"] # only these MCP tools
  - type: filesystem
    tools: ["read_file", "search_files_content"] # only these filesystem tools
```

## Tool Instructions

Add custom instructions that are injected into the agent's context when a toolset is loaded:

```yaml
toolsets:
  - type: mcp
    ref: docker:github-official
    instruction: |
      Use these tools to manage GitHub issues.
      Always check for existing issues before creating new ones.
```

<div class="callout callout-tip">
<div class="callout-title">💡 See also
</div>
  <p>For connecting to 50+ cloud services via remote MCP with OAuth, see <a href="/features/remote-mcp/">Remote MCP Servers</a>. For RAG (document retrieval), see <a href="/features/rag/">RAG</a>. For full tool config reference, see <a href="/configuration/tools/">Tool Config</a>.</p>

</div>
