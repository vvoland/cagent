---
title: "Agents"
description: "Agents are the core building blocks of cagent. Each agent is an AI-powered entity with a model, instructions, tools, and optional sub-agents."
permalink: /concepts/agents/
---

# Agents

_Agents are the core building blocks of cagent. Each agent is an AI-powered entity with a model, instructions, tools, and optional sub-agents._

## What is an Agent?

An agent in cagent is defined by:

- **Model** — The AI model powering it (e.g., Claude, GPT-4o, Gemini). See [Models](/concepts/models/).
- **Description** — A brief summary of what the agent does (used by other agents for delegation)
- **Instruction** — The system prompt that defines the agent's behavior and personality
- **Tools** — Capabilities like filesystem access, shell commands, or external APIs
- **Sub-agents** — Other agents it can delegate tasks to

```yaml
agents:
  root:
    model: anthropic/claude-sonnet-4-0
    description: Expert software developer
    instruction: |
      You are an expert developer. Write clean, efficient code
      and explain your reasoning step by step.
    toolsets:
      - type: filesystem
      - type: shell
      - type: think
```

## The Root Agent

Every cagent configuration has a **root agent** — the entry point that receives user messages. In a single-agent setup, this is the only agent. In a multi-agent setup, the root agent acts as a coordinator, delegating tasks to specialized sub-agents.

<div class="callout callout-info">
<div class="callout-title">ℹ️ Naming
</div>
  <p>The first agent defined in your YAML (or the one named <code>root</code>) is the root agent by default. You can also specify which agent to start with using <code>cagent run config.yaml -a agent_name</code>.</p>

</div>

## Agent Properties

| Property               | Type    | Required | Description                                                    |
| ---------------------- | ------- | -------- | -------------------------------------------------------------- |
| `model`                | string  | ✓        | Model reference (inline like `openai/gpt-4o` or a named model) |
| `description`          | string  | ✓        | What the agent does — used by other agents for delegation      |
| `instruction`          | string  | ✓        | System prompt defining behavior                                |
| `toolsets`             | array   | ✗        | List of tool configurations                                    |
| `sub_agents`           | array   | ✗        | Names of agents this agent can delegate to                     |
| `fallback`             | object  | ✗        | Fallback model configuration for resilience                    |
| `add_date`             | boolean | ✗        | Include current date in context                                |
| `add_environment_info` | boolean | ✗        | Include OS, working directory, git info in context             |
| `max_iterations`       | int     | ✗        | Max tool-calling loops (default: unlimited)                    |
| `commands`             | object  | ✗        | Named prompts callable via `/command`                          |
| `skills`               | boolean | ✗        | Enable skill discovery and loading                             |

## Model Fallbacks

Agents can automatically fail over to alternative models when the primary model is unavailable:

```yaml
agents:
  root:
    model: anthropic/claude-sonnet-4-0
    fallback:
      models:
        - openai/gpt-4o
        - google/gemini-2.5-flash
      retries: 2 # retries per model for 5xx errors
      cooldown: 1m # stick with fallback after 429
```

## Named Commands

Define reusable prompts that can be invoked as commands:

```yaml
agents:
  root:
    model: openai/gpt-4o
    instruction: You are a helpful assistant.
    commands:
      df: "Check how much free space I have on my disk"
      greet: "Say hello to ${env.USER}"
```

```bash
# Run a named command
$ cagent run agent.yaml /df
$ cagent run agent.yaml /greet
```

Commands support environment variable interpolation using JavaScript template literal syntax. Undefined variables expand to empty strings.

## Default Agent

Running `cagent run` without a config file uses a built-in default agent. This is a capable general-purpose agent for quick tasks without needing any configuration.

```bash
# Use the default agent
$ cagent run

# Override the default with an alias
$ cagent alias add default /path/to/my-agent.yaml
$ cagent run  # now runs your custom agent
```

<div class="callout callout-tip">
<div class="callout-title">💡 See also
</div>
  <p>For reusable task-specific instructions, see <a href="/features/skills/">Skills</a>. For multi-agent patterns, see <a href="/concepts/multi-agent/">Multi-Agent</a>. For full config reference, see <a href="/configuration/agents/">Agent Config</a>.</p>

</div>
