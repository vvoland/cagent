---
title: "Anthropic"
description: "Use Claude Sonnet 4, Claude Sonnet 4.5, and other Anthropic models with cagent."
permalink: /providers/anthropic/
---

# Anthropic

_Use Claude Sonnet 4, Claude Sonnet 4.5, and other Anthropic models with cagent._

## Setup

```bash
# Set your API key
export ANTHROPIC_API_KEY="sk-ant-..."
```

## Configuration

### Inline

```yaml
agents:
  root:
    model: anthropic/claude-sonnet-4-0
```

### Named Model

```yaml
models:
  claude:
    provider: anthropic
    model: claude-sonnet-4-0
    max_tokens: 64000
```

## Available Models

| Model               | Best For                            |
| ------------------- | ----------------------------------- |
| `claude-sonnet-4-5` | Most capable, extended thinking     |
| `claude-sonnet-4-0` | Strong coding, balanced performance |

## Thinking Budget

Anthropic uses integer token budgets (1024–32768). Defaults to 8192 with interleaved thinking enabled:

```yaml
models:
  claude-deep:
    provider: anthropic
    model: claude-sonnet-4-5
    thinking_budget: 16384 # must be < max_tokens
```

## Interleaved Thinking

Enabled by default. Allows tool calls during model reasoning for more integrated problem-solving:

```yaml
models:
  claude:
    provider: anthropic
    model: claude-sonnet-4-5
    provider_opts:
      interleaved_thinking: false # disable if needed
```

<div class="callout callout-info">
<div class="callout-title">ℹ️ Note
</div>
  <p>Anthropic thinking budget values below 1024 or greater than or equal to <code>max_tokens</code> are ignored (a warning is logged).</p>

</div>
