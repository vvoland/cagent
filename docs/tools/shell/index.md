---
title: "Shell Tool"
description: "Execute arbitrary shell commands in the user's environment."
permalink: /tools/shell/
---

# Shell Tool

_Execute arbitrary shell commands in the user's environment._

## Overview

The shell tool allows agents to execute arbitrary shell commands. This is one of the most powerful tools — it lets agents run builds, install dependencies, query APIs, and interact with the system. Each call runs in a fresh, isolated shell session — no state persists between calls.

Commands have a default 30-second timeout and require user confirmation unless `--yolo` is used.

## Configuration

```yaml
toolsets:
  - type: shell
```

### Options

| Property | Type   | Description                                         |
| -------- | ------ | --------------------------------------------------- |
| `env`    | object | Environment variables to set for all shell commands |

### Custom Environment Variables

```yaml
toolsets:
  - type: shell
    env:
      MY_VAR: "value"
      PATH: "${PATH}:/custom/bin"
```

<div class="callout callout-warning" markdown="1">
<div class="callout-title">⚠️ Safety
</div>
  <p>The shell tool gives agents full access to the system shell. Always set <code>max_iterations</code> on agents that use the shell tool to prevent infinite loops. A value of 20–50 is typical for development agents. Use <a href="{{ '/configuration/sandbox/' | relative_url }}">Sandbox Mode</a> for additional isolation.</p>
</div>

<div class="callout callout-info" markdown="1">
<div class="callout-title">ℹ️ Tool Confirmation
</div>
  <p>By default, docker-agent asks for user confirmation before executing shell commands. Use <code>--yolo</code> to auto-approve all tool calls.</p>
</div>
