---
title: "Docker Model Runner"
description: "Run AI models locally with Docker — no API keys, no costs, full data privacy."
permalink: /providers/dmr/
---

# Docker Model Runner

_Run AI models locally with Docker — no API keys, no costs, full data privacy._

## Overview

Docker Model Runner (DMR) lets you run open-source AI models directly on your machine. Models run in Docker, so there's no API key needed and no data leaves your computer.

<div class="callout callout-tip">
<div class="callout-title">💡 No API key needed
</div>
  <p>DMR runs models locally — your data never leaves your machine. Great for development, sensitive data, or offline use.</p>

</div>

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) with the Model Runner feature enabled
- Verify with: `docker model status --json`

## Configuration

### Inline

```yaml
agents:
  root:
    model: dmr/ai/qwen3
```

### Named Model

```yaml
models:
  local:
    provider: dmr
    model: ai/qwen3
    max_tokens: 8192
```

## Available Models

Any model available through Docker Model Runner can be used. Common options:

| Model         | Description                                           |
| ------------- | ----------------------------------------------------- |
| `ai/qwen3`    | Qwen 3 — versatile, good for coding and general tasks |
| `ai/llama3.2` | Llama 3.2 — Meta's open-source model                  |

## Runtime Flags

Pass flags to the underlying inference runtime (e.g., llama.cpp) using `provider_opts.runtime_flags`:

```yaml
models:
  local:
    provider: dmr
    model: ai/qwen3
    max_tokens: 8192
    provider_opts:
      runtime_flags: ["--ngl=33", "--top-p=0.9"]
```

Runtime flags also accept a single string:

```yaml
provider_opts:
  runtime_flags: "--ngl=33 --top-p=0.9"
```

## Parameter Mapping

docker-agent model config fields map to llama.cpp flags automatically:

| Config              | llama.cpp Flag        |
| ------------------- | --------------------- |
| `temperature`       | `--temp`              |
| `top_p`             | `--top-p`             |
| `frequency_penalty` | `--frequency-penalty` |
| `presence_penalty`  | `--presence-penalty`  |
| `max_tokens`        | `--context-size`      |

`runtime_flags` always take priority over derived flags on conflict.

## Speculative Decoding

Use a smaller draft model to predict tokens ahead for faster inference:

```yaml
models:
  fast-local:
    provider: dmr
    model: ai/qwen3:14B
    max_tokens: 8192
    provider_opts:
      speculative_draft_model: ai/qwen3:0.6B-F16
      speculative_num_tokens: 16
      speculative_acceptance_rate: 0.8
```

## Custom Endpoint

If `base_url` is omitted, docker-agent auto-discovers the DMR endpoint. To set manually:

```yaml
models:
  local:
    provider: dmr
    model: ai/qwen3
    base_url: http://127.0.0.1:12434/engines/llama.cpp/v1
```

## Troubleshooting

- **Plugin not found:** Ensure Docker Model Runner is enabled in Docker Desktop. docker-agent will fall back to the default URL.
- **Endpoint empty:** Verify the Model Runner is running with `docker model status --json`.
- **Performance:** Use `runtime_flags` to tune GPU layers (`--ngl`) and thread count (`--threads`).
