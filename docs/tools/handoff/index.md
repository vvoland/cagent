---
title: "Handoff Tool"
description: "Delegate tasks to remote agents via the A2A protocol."
permalink: /tools/handoff/
---

# Handoff Tool

_Delegate tasks to remote agents via the A2A protocol._

## Overview

The handoff tool lets agents delegate tasks to remote agents via the A2A (Agent-to-Agent) protocol. Use this to connect to agents running as separate services or on different machines.

## Configuration

```yaml
toolsets:
  - type: handoff
    name: research_agent
    description: Specialized research agent
    url: "http://localhost:8080/a2a"
    timeout: 5m
```

## Properties

| Property      | Type   | Required | Description                          |
| ------------- | ------ | -------- | ------------------------------------ |
| `name`        | string | ✓        | Tool name for delegation             |
| `description` | string | ✗        | Description shown to the agent       |
| `url`         | string | ✓        | A2A server endpoint URL              |
| `timeout`     | string | ✗        | Request timeout (default: `5m`)      |

<div class="callout callout-tip" markdown="1">
<div class="callout-title">💡 See also
</div>
  <p>For full details on the A2A protocol and serving agents as A2A endpoints, see <a href="{{ '/features/a2a/' | relative_url }}">A2A Protocol</a>.</p>
</div>
