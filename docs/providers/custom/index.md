---
title: "Custom Providers"
description: "Connect docker-agent to any OpenAI-compatible API endpoint — without modifying docker-agent's source code."
permalink: /providers/custom/
---

# Custom Providers

_Connect docker-agent to any OpenAI-compatible API endpoint — without modifying docker-agent's source code._

## Overview

The `providers` section in your agent YAML lets you define custom providers that work with any OpenAI-compatible API. This is useful for:

- Self-hosted models (vLLM, Ollama, LocalAI, etc.)
- API proxies and routers (Requesty, LiteLLM, etc.)
- Enterprise deployments with custom endpoints
- Any service with an OpenAI-compatible chat completions API

<div class="callout callout-info">
<div class="callout-title">ℹ️ Works with any OpenAI-compatible API
</div>
  <p>If a service supports the <code>/v1/chat/completions</code> endpoint, you can use it with docker-agent. No source code changes needed.</p>

</div>

## Configuration

```yaml
providers:
  my_provider:
    api_type: openai_chatcompletions # or openai_responses
    base_url: https://api.example.com/v1
    token_key: MY_API_KEY # env var name

models:
  my_model:
    provider: my_provider
    model: gpt-4o
    max_tokens: 32768

agents:
  root:
    model: my_model
    instruction: You are a helpful assistant.
```

## Provider Properties

| Property    | Description                                                | Default                  |
| ----------- | ---------------------------------------------------------- | ------------------------ |
| `api_type`  | API schema: `openai_chatcompletions` or `openai_responses` | `openai_chatcompletions` |
| `base_url`  | Base URL for the API endpoint                              | —                        |
| `token_key` | Name of the environment variable containing the API token  | —                        |

## Shorthand Syntax

Once a custom provider is defined, you can use the shorthand `provider/model` syntax:

```yaml
agents:
  root:
    model: my_provider/gpt-4o-mini # uses the provider's base_url and token
```

## API Types

- **`openai_chatcompletions`** — Standard OpenAI Chat Completions API. Works with most OpenAI-compatible endpoints.
- **`openai_responses`** — OpenAI Responses API. For newer models that require the Responses API format.

## Examples

### vLLM / Ollama

```yaml
providers:
  local_llm:
    api_type: openai_chatcompletions
    base_url: http://localhost:8000/v1

agents:
  root:
    model: local_llm/llama-3.1-8b
```

### API Router (Requesty, LiteLLM)

```yaml
providers:
  router:
    api_type: openai_chatcompletions
    base_url: https://router.requesty.ai/v1
    token_key: REQUESTY_API_KEY

agents:
  root:
    model: router/anthropic/claude-sonnet-4-0
```

### Azure OpenAI

```yaml
models:
  azure_model:
    provider: azure
    model: gpt-4o
    base_url: https://your-llm.openai.azure.com
    provider_opts:
      api_version: 2024-12-01-preview
```

## How It Works

When you reference a custom provider:

1. The provider's `base_url` is applied to the model (if not already set)
2. The provider's `token_key` is applied to the model (if not already set)
3. The provider's `api_type` is stored in `provider_opts.api_type`
4. The model is used with the appropriate API client
