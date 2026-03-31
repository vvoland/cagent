---
title: "Sandbox Mode"
description: "Run agents in an isolated Docker container for enhanced security."
permalink: /configuration/sandbox/
---

# Sandbox Mode

_Run agents in an isolated Docker container for enhanced security._

## Overview

Sandbox mode runs the entire agent inside a Docker container instead of directly on the host system. This provides an additional layer of isolation, limiting the potential impact of unintended or malicious commands.

<div class="callout callout-info" markdown="1">
<div class="callout-title">ℹ️ Requirements
</div>
  <p>Sandbox mode requires Docker to be installed and running on the host system.</p>

</div>

## Usage

Enable sandbox mode with the `--sandbox` flag on the `docker agent run` command:

```bash
docker agent run --sandbox agent.yaml
```

This runs the agent inside a Docker container with the current working directory mounted.

## Example

```yaml
# agent.yaml
agents:
  root:
    model: openai/gpt-4o
    description: Agent with sandboxed shell
    instruction: You are a helpful assistant.
    toolsets:
      - type: shell
```

```bash
docker agent run --sandbox agent.yaml
```

## How It Works

1. When `--sandbox` is specified, docker-agent launches a Docker container
2. The current working directory is mounted into the container
3. All agent tools (shell, filesystem, etc.) operate inside the container
4. When the session ends, the container is automatically stopped and removed

<div class="callout callout-warning" markdown="1">
<div class="callout-title">⚠️ Limitations
</div>

- Container starts fresh each session (no persistence between sessions)

</div>