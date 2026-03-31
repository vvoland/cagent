---
title: "Transfer Task Tool"
description: "Delegate tasks to sub-agents in multi-agent setups."
permalink: /tools/transfer-task/
---

# Transfer Task Tool

_Delegate tasks to sub-agents in multi-agent setups._

## Overview

The `transfer_task` tool allows an agent to delegate tasks to specialized sub-agents and receive their results. This is the core mechanism for multi-agent orchestration.

**You don't need to add it manually** — it's automatically available when an agent has `sub_agents` configured.

## Configuration

The tool is enabled implicitly when `sub_agents` is set:

```yaml
agents:
  coordinator:
    model: openai/gpt-4o
    description: Coordinates work across specialists
    instruction: Analyze requests and delegate to the right specialist.
    sub_agents: [developer, researcher]

  developer:
    model: anthropic/claude-sonnet-4-0
    description: Expert software developer
    instruction: Write clean, production-ready code.
    toolsets:
      - type: filesystem
      - type: shell

  researcher:
    model: openai/gpt-4o
    description: Web researcher
    instruction: Search for information online.
    toolsets:
      - type: mcp
        ref: docker:duckduckgo
```

The coordinator agent automatically gets a `transfer_task` tool that can delegate to `developer` or `researcher`.

<div class="callout callout-tip" markdown="1">
<div class="callout-title">💡 See also
</div>
  <p>For parallel task delegation, see <a href="{{ '/tools/background-agents/' | relative_url }}">Background Agents</a>. For multi-agent patterns, see <a href="{{ '/concepts/multi-agent/' | relative_url }}">Multi-Agent</a>.</p>
</div>
