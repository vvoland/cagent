---
title: "A2A Protocol"
description: "Expose cagent agents via Google's Agent-to-Agent (A2A) protocol for interoperability with other agent frameworks."
permalink: /features/a2a/
---

# A2A Protocol

_Expose cagent agents via Google's Agent-to-Agent (A2A) protocol for interoperability with other agent frameworks._

## Overview

The `cagent serve a2a` command starts an A2A server that exposes your agents using the [A2A protocol](https://google.github.io/A2A/). This enables communication between cagent and other agent frameworks that support A2A.

<div class="callout callout-warning">
<div class="callout-title">⚠️ Early support
</div>
  <p>A2A support is functional but still evolving. Tool calls, artifacts, and memory features have limited A2A integration. See limitations below.</p>

</div>

## Usage

```bash
# Start A2A server for an agent
$ cagent serve a2a ./agent.yaml

# Specify a custom address
$ cagent serve a2a ./agent.yaml --listen 127.0.0.1:9000

# Use an agent from the catalog
$ cagent serve a2a agentcatalog/pirate
```

## Features

- **Auto port selection** — Picks an available port if not specified
- **Agent card** — Provides standard A2A agent metadata
- **Full cagent features** — Supports all tools, models, and gateway features
- **Multiple sources** — Load agents from files or the agent catalog

<div class="callout callout-tip">
<div class="callout-title">💡 See also
</div>
  <p>For exposing agents via MCP instead, see <a href="/features/mcp-mode/">MCP Mode</a>. For stdio-based integration, see <a href="/features/acp/">ACP</a>. For the HTTP API, see <a href="/features/api-server/">API Server</a>.</p>

</div>

## Current Limitations

- Tool calls are handled internally, not exposed as separate A2A events
- A2A artifact support not yet integrated
- A2A memory features not yet integrated
- Multi-agent (sub-agent) scenarios need further work
