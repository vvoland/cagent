---
title: "OpenAI"
description: "Use GPT-4o, GPT-5, GPT-5-mini, and other OpenAI models with cagent."
permalink: /providers/openai/
---

# OpenAI

_Use GPT-4o, GPT-5, GPT-5-mini, and other OpenAI models with cagent._

## Setup

```bash
# Set your API key
export OPENAI_API_KEY="sk-..."
```

## Configuration

### Inline

```yaml
agents:
  root:
    model: openai/gpt-4o
```

### Named Model

```yaml
models:
  gpt:
    provider: openai
    model: gpt-4o
    temperature: 0.7
    max_tokens: 4000
```

## Available Models

| Model         | Best For                             |
| ------------- | ------------------------------------ |
| `gpt-5`       | Most capable, complex reasoning      |
| `gpt-5-mini`  | Fast, cost-effective, good reasoning |
| `gpt-4o`      | Multimodal, balanced performance     |
| `gpt-4o-mini` | Cheapest, fast for simple tasks      |

Find more model names at [modelname.ai](https://modelname.ai/).

## Thinking Budget

OpenAI uses effort level strings:

```yaml
models:
  gpt-thinking:
    provider: openai
    model: gpt-5-mini
    thinking_budget: low # minimal | low | medium (default) | high
```

<div class="callout callout-tip">
<div class="callout-title">💡 Custom endpoints
</div>
  <p>Use <code>base_url</code> for proxies and OpenAI-compatible services. See <a href="/providers/custom/">Custom Providers</a> for full setup.</p>

</div>

## Custom Endpoint

Use `base_url` to connect to OpenAI-compatible APIs:

```yaml
models:
  custom:
    provider: openai
    model: gpt-4o
    base_url: https://your-proxy.example.com/v1
```
