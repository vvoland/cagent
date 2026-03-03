---
title: "ACP (Agent Client Protocol)"
description: "Expose cagent agents via the Agent Client Protocol for integration with ACP-compatible hosts like VS Code, IDEs, and other developer tools."
permalink: /features/acp/
---

# ACP (Agent Client Protocol)

_Expose cagent agents via the Agent Client Protocol for integration with ACP-compatible hosts like VS Code, IDEs, and other developer tools._

## Overview

The `cagent acp` command starts an ACP server that communicates over **stdio** (standard input/output). This makes it ideal for integration with editors, IDEs, and other tools that spawn agent processes — the host sends JSON-RPC messages to cagent's stdin and reads responses from stdout.

ACP is built on the [ACP Go SDK](https://github.com/coder/acp-go-sdk) and provides a standardized way for client applications to interact with AI agents.

<div class="callout callout-info">
<div class="callout-title">ℹ️ ACP vs A2A vs MCP
</div>
  **ACP** connects an agent to a *host application* (IDE, CLI tool) via stdio. **A2A** connects *agents to other agents* over HTTP. **MCP** exposes agents as *tools* for other MCP clients. Choose based on your integration target.

</div>

## Usage

```bash
# Start ACP server on stdio
$ cagent acp ./agent.yaml

# With a multi-agent team config
$ cagent acp ./team.yaml

# From the agent catalog
$ cagent acp agentcatalog/pirate

# With a custom session database
$ cagent acp ./agent.yaml --session-db ./my-sessions.db
```

## How It Works

1. The host application spawns `cagent acp agent.yaml` as a child process
2. Communication happens over **stdin/stdout** using the ACP protocol
3. The host sends user messages, cagent processes them through the agent
4. Agent responses, tool calls, and events stream back to the host
5. Sessions are persisted in a SQLite database for continuity

```bash
# Conceptual flow:
Host Application
  └── spawns: cagent acp agent.yaml
        ├── stdin  ← JSON-RPC requests from host
        └── stdout → JSON-RPC responses to host
```

## Features

- **Stdio transport** — No network ports needed; ideal for subprocess integration
- **Session persistence** — SQLite-backed sessions survive process restarts
- **Full agent support** — All cagent features work: tools, multi-agent, model fallbacks
- **Multi-agent configs** — Team configurations with sub-agents work transparently
- **Filesystem operations** — Agents can read/write files relative to the host's working directory

## CLI Flags

```bash
cagent acp <agent-file>|<registry-ref> [flags]
```

| Flag               | Default                | Description                         |
| ------------------ | ---------------------- | ----------------------------------- |
| `-s, --session-db` | `~/.cagent/session.db` | Path to the SQLite session database |

## Integration Example

A host application would spawn cagent as a subprocess and communicate via the ACP protocol:

```javascript
// Pseudocode for an IDE extension
const child = spawn("cagent", ["acp", "./agent.yaml"]);

// Send a message to the agent
child.stdin.write(
  JSON.stringify({
    jsonrpc: "2.0",
    method: "agent/run",
    params: { message: "Explain this code" },
  }),
);

// Read responses
child.stdout.on("data", (data) => {
  const response = JSON.parse(data);
  // Handle agent response, tool calls, etc.
});
```

<div class="callout callout-tip">
<div class="callout-title">💡 When to use ACP
</div>
  <p>Use ACP when building **IDE integrations**, **editor plugins**, or any tool that wants to embed a cagent agent as a subprocess. For HTTP-based integrations, use the <a href="/features/api-server/">API Server</a> instead.</p>

</div>

<div class="callout callout-info">
<div class="callout-title">ℹ️ See also
</div>
  <p>For HTTP-based agent access, see the <a href="/features/api-server/">API Server</a>. For agent-to-agent communication, see <a href="/features/a2a/">A2A Protocol</a>. For exposing agents as MCP tools, see <a href="/features/mcp-mode/">MCP Mode</a>.</p>

</div>
