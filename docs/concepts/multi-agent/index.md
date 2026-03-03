---
title: "Multi-Agent Systems"
description: "Build teams of specialized agents that collaborate and delegate tasks to each other."
permalink: /concepts/multi-agent/
---

# Multi-Agent Systems

_Build teams of specialized agents that collaborate and delegate tasks to each other._

## Why Multi-Agent?

Complex tasks benefit from specialization. Instead of one monolithic agent trying to do everything, you can create a **team** of focused agents:

- A **coordinator** that understands the overall goal and delegates
- A **developer** that writes code with filesystem and shell access
- A **reviewer** that checks code quality
- A **researcher** that searches the web for information

Each agent has its own model, tools, and instructions — optimized for its specific role.

## How Delegation Works

Agents delegate tasks using the built-in `transfer_task` tool, which is automatically available to any agent with sub-agents. This smart delegation means agents can automatically route tasks to the most suitable specialist.

1. **User** sends a message to the root agent
2. **Root agent** analyzes the request and decides which sub-agent should handle it
3. **Root agent** calls `transfer_task` with the target agent, task description, and expected output
4. **Sub-agent** processes the task in its own agentic loop using its tools
5. **Results** flow back to the root agent, which responds to the user

```bash
# The transfer_task tool call looks like:
transfer_task(
  agent="developer",
  task="Create a REST API endpoint for user authentication",
  expected_output="Working Go code with tests"
)
```

<div class="callout callout-info">
<div class="callout-title">ℹ️ Auto-Approved
</div>
  <p>Unlike other tools, <code>transfer_task</code> is always auto-approved — no user confirmation needed. This allows seamless delegation between agents.</p>

</div>

## Example: Development Team

```yaml
agents:
  root:
    model: anthropic/claude-sonnet-4-0
    description: Technical lead coordinating development
    instruction: |
      You are a technical lead managing a development team.
      Analyze requests and delegate to the right specialist.
      Ensure quality by reviewing results before responding.
    sub_agents: [developer, reviewer, tester]
    toolsets:
      - type: think

  developer:
    model: anthropic/claude-sonnet-4-0
    description: Expert software developer
    instruction: |
      You are an expert developer. Write clean, efficient code
      and follow best practices.
    toolsets:
      - type: filesystem
      - type: shell
      - type: think

  reviewer:
    model: openai/gpt-4o
    description: Code review specialist
    instruction: |
      You review code for quality, security, and maintainability.
      Provide actionable feedback.
    toolsets:
      - type: filesystem

  tester:
    model: openai/gpt-4o
    description: Quality assurance engineer
    instruction: |
      You write tests and ensure software quality. Run tests
      and report results.
    toolsets:
      - type: shell
      - type: todo
```

## Example: Research Team

```yaml
agents:
  root:
    model: anthropic/claude-sonnet-4-0
    description: Research coordinator
    instruction: |
      Coordinate research tasks. Delegate web searches to
      the researcher and writing to the writer.
    sub_agents: [researcher, writer]
    toolsets:
      - type: think

  researcher:
    model: openai/gpt-4o
    description: Web researcher
    instruction: Search the web and gather information.
    toolsets:
      - type: mcp
        ref: docker:duckduckgo
      - type: memory
        path: ./research.db

  writer:
    model: anthropic/claude-sonnet-4-0
    description: Content writer
    instruction: Write clear, well-structured content.
    toolsets:
      - type: filesystem
```

## Multi-Model Teams

A key advantage of multi-agent systems is using different models for different roles — picking the best model for each job:

```yaml
models:
  fast:
    provider: openai
    model: gpt-5-mini
    temperature: 0.2 # precise

  creative:
    provider: openai
    model: gpt-4o
    temperature: 0.8 # creative

  local:
    provider: dmr
    model: ai/qwen3 # runs locally, no API cost

agents:
  analyst:
    model: fast # cheap and fast for analysis
  writer:
    model: creative # creative for content
  helper:
    model: local # free for simple tasks
```

## Shared Tools

Tools like `todo` can be shared between agents for collaborative task tracking:

```yaml
toolsets:
  - type: todo
    shared: true # all agents see the same todo list
```

## Best Practices

- **Keep agents focused** — Each agent should have a clear, narrow role
- **Write clear descriptions** — The coordinator uses descriptions to decide who to delegate to
- **Give minimal tools** — Only give each agent the tools it needs for its specific role
- **Use the think tool** — Give coordinators the think tool so they reason about delegation
- **Use the right model** — Use capable models for complex reasoning, cheap models for simple tasks

<div class="callout callout-info">
<div class="callout-title">ℹ️ Beyond cagent
</div>
  <p>For interoperability with other agent frameworks, cagent supports the <a href="/features/a2a/">A2A protocol</a> and can expose agents via <a href="/features/mcp-mode/">MCP Mode</a>.</p>

</div>
