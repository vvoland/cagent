---
title: "Sandbox Mode"
description: "Run shell commands in an isolated Docker container for enhanced security."
permalink: /configuration/sandbox/
---

# Sandbox Mode

_Run shell commands in an isolated Docker container for enhanced security._

## Overview

Sandbox mode runs shell tool commands inside a Docker container instead of directly on the host system. This provides an additional layer of isolation, limiting the potential impact of unintended or malicious commands.

<div class="callout callout-info">
<div class="callout-title">ℹ️ Requirements
</div>
  <p>Sandbox mode requires Docker to be installed and running on the host system.</p>

</div>

## Configuration

```yaml
agents:
  root:
    model: openai/gpt-4o
    description: Agent with sandboxed shell
    instruction: You are a helpful assistant.
    toolsets:
      - type: shell
        sandbox:
          image: alpine:latest # Docker image to use
          paths: # Directories to mount
            - "." # Current directory (read-write)
            - "/data:ro" # Read-only mount
```

## Properties

| Property | Type   | Default         | Description                                   |
| -------- | ------ | --------------- | --------------------------------------------- |
| `image`  | string | `alpine:latest` | Docker image to use for the sandbox container |
| `paths`  | array  | `[]`            | Host paths to mount into the container        |

## Path Mounting

Paths can be specified with optional access modes:

| Format       | Description                                     |
| ------------ | ----------------------------------------------- |
| `/path`      | Mount with read-write access (default)          |
| `/path:rw`   | Explicitly read-write                           |
| `/path:ro`   | Read-only mount                                 |
| `.`          | Current working directory                       |
| `./relative` | Relative path (resolved from working directory) |

Paths are mounted at the same location inside the container as on the host, so file paths in commands work the same way.

## Example: Development Agent

```yaml
agents:
  developer:
    model: anthropic/claude-sonnet-4-0
    description: Development agent with sandboxed shell
    instruction: |
      You are a software developer. Use the shell tool to run
      build commands and tests. Your shell runs in a sandbox.
    toolsets:
      - type: shell
        sandbox:
          image: node:20-alpine # Node.js environment
          paths:
            - "." # Project directory
            - "/tmp:rw" # Temp directory for builds
      - type: filesystem
```

## How It Works

1. When the agent first uses the shell tool, cagent starts a Docker container
2. The container runs with the specified image and mounted paths
3. Shell commands execute inside the container via `docker exec`
4. The container persists for the session (commands share state)
5. When the session ends, the container is automatically stopped and removed

## Container Configuration

Sandbox containers are started with these Docker options:

- `--rm` — Automatically remove when stopped
- `--init` — Use init process for proper signal handling
- `--network host` — Share host network (commands can access network)
- Environment variables from host are forwarded to container

## Orphan Container Cleanup

If cagent crashes or is killed, sandbox containers may be left running. cagent automatically cleans up orphaned containers from previous runs when it starts. Containers are identified by labels and the PID of the cagent process that created them.

## Choosing an Image

Select a Docker image that has the tools your agent needs:

| Use Case               | Suggested Image      |
| ---------------------- | -------------------- |
| General scripting      | `alpine:latest`      |
| Node.js development    | `node:20-alpine`     |
| Python development     | `python:3.12-alpine` |
| Go development         | `golang:1.23-alpine` |
| Full Linux environment | `ubuntu:24.04`       |

<div class="callout callout-tip">
<div class="callout-title">💡 Custom Images
</div>
  <p>For complex setups, build a custom Docker image with all required tools pre-installed. This avoids installation time during agent execution.</p>

</div>

<div class="callout callout-warning">
<div class="callout-title">⚠️ Limitations
</div>

- Only the <code>shell</code> tool runs in the sandbox; other tools (filesystem, MCP) run on the host
- Host network access means network-based attacks are still possible
- Mounted paths are accessible according to their access mode
- Container starts fresh each session (no persistence between sessions)

</div>

## Combining with Permissions

For defense in depth, combine sandbox mode with [permissions](/configuration/permissions/):

```yaml
agents:
  root:
    model: openai/gpt-4o
    description: Secure development agent
    instruction: You are a helpful assistant.
    toolsets:
      - type: shell
        sandbox:
          image: node:20-alpine
          paths:
            - ".:rw"
      - type: filesystem

permissions:
  allow:
    - "shell:cmd=npm*"
    - "shell:cmd=node*"
    - "shell:cmd=ls*"
  deny:
    - "shell:cmd=sudo*"
    - "shell:cmd=curl*"
    - "shell:cmd=wget*"
```
