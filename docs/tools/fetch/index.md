---
title: "Fetch Tool"
description: "Make HTTP requests to external APIs and web services."
permalink: /tools/fetch/
---

# Fetch Tool

_Make HTTP requests to external APIs and web services._

## Overview

The fetch tool lets agents make HTTP requests (GET, POST, PUT, DELETE, etc.) to external APIs. The agent can read web pages, call REST APIs, download data, and interact with web services.

## Configuration

```yaml
toolsets:
  - type: fetch
```

### Options

| Property  | Type | Default | Description                |
| --------- | ---- | ------- | -------------------------- |
| `timeout` | int  | `30`    | Request timeout in seconds |

### Custom Timeout

```yaml
toolsets:
  - type: fetch
    timeout: 60
```

<div class="callout callout-tip">
<div class="callout-title">💡 Fetch vs. API Tool
</div>
  <p>The fetch tool gives the agent full control over HTTP requests at runtime. The <a href="{{ '/tools/api/' | relative_url }}">API tool</a> lets you predefine specific API calls as named tools with typed parameters. Use fetch for general-purpose HTTP access; use the API tool for well-known endpoints you want to expose as structured tools.</p>
</div>
