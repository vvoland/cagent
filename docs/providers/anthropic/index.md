---
title: "Anthropic"
description: "Use Claude Sonnet 4, Claude Sonnet 4.5, and other Anthropic models with docker-agent."
permalink: /providers/anthropic/
---

# Anthropic

_Use Claude Sonnet 4, Claude Sonnet 4.5, and other Anthropic models with docker-agent._

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

## Task Budget

`task_budget` caps the **total** number of tokens the model may spend across a
multi-step agentic task — combined thinking, tool calls, and final output. It
is forwarded as
[`output_config.task_budget`](https://platform.claude.com/docs/en/about-claude/models/whats-new-claude-4-7)
and is ideal for letting long-running agents self-regulate effort without
tightening `max_tokens` on every call.

docker-agent automatically attaches the required `task-budgets-2026-03-13`
beta header whenever this field is set. You can configure `task_budget` on
**any** Claude model — docker-agent never gates it by model name. At the time
of writing, only **Claude Opus 4.7** actually honors the field; other Claude
models (Sonnet 4.5, Opus 4.5 / 4.6, etc.) are expected to reject requests
that include it. Check the Anthropic release notes linked above for the
current list of supported models.

```yaml
models:
  opus:
    provider: anthropic
    model: claude-opus-4-7
    task_budget: 128000 # integer shorthand → { type: tokens, total: 128000 }
    thinking_budget: adaptive
```

Object form (forward-compatible with future budget types):

```yaml
  opus:
    provider: anthropic
    model: claude-opus-4-7
    task_budget:
      type: tokens
      total: 128000
```

See the full schema on the [Model Configuration]({{ '/configuration/models/#task-budget' | relative_url }}) page.

<div class="callout callout-info" markdown="1">
<div class="callout-title">ℹ️ Note
</div>
  <p>Anthropic thinking budget values below 1024 or greater than or equal to <code>max_tokens</code> are ignored (a warning is logged).</p>

</div>
