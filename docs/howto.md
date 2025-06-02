# How to Create and Configure Agents

This guide will help you create and configure intelligent agents using YAML.
Following the steps below, you'll learn how to build agents ranging from simple
conversational bots to complex systems with tools and sub-agents.

## Prerequisites

- Basic understanding of YAML syntax
- Docker installed (for running certain agent tools)
- Understanding of the agent configuration structure

## Creating a Basic Agent

### Step 1: Define the Agent Structure

Create a YAML file with the following basic structure:

```yaml
agents:
  root:
    name: my_agent
    model: openai
    description: A brief description of your agent
    instruction: |
      Detailed instructions for your agent's behavior
```

### Step 2: Configure the Agent's Personality

The `instruction` field defines how your agent will behave. For example, to
create a pirate-speaking agent:

```yaml
agents:
  root:
    name: pirate
    model: openai
    description: An agent that talks like a pirate
    instruction: |
      Always answer by talking like a pirate.
```

### Step 3: Specify the Language Model

Define which AI model your agent will use:

```yaml
models:
  openai:
    type: openai
    model: gpt-4o
    temperature: 0.7
    max_tokens: 1500
```

## Adding Advanced Capabilities

### Integrating Tools

Tools extend your agent's capabilities, allowing it to interact with external
systems:

```yaml
agents:
  root:
    # Basic configuration...
    tools:
      - type: mcp
        command: docker
        args:
          [
            "run",
            "-i",
            "--rm",
            "alpine/socat",
            "STDIO",
            "TCP:host.docker.internal:8811",
          ]
```

Example: Adding an Airbnb search tool:

```yaml
tools:
  - type: mcp
    command: npx
    args: ["-y", "@openbnb/mcp-server-airbnb", "--ignore-robots-txt"]
```

### Creating Sub-Agents

Sub-agents help distribute specialized tasks:

```yaml
agents:
  root:
    # Basic configuration...
    sub_agents:
      - containerize
      - optimize_dockerfile

  containerize:
    name: containerize
    model: openai
    description: |
      You are a helpful assistant for containerizing applications.
    instruction: |
      You are an expert in Docker.
      # Additional instructions...
```

### Implementing the 'Think' Tool

The 'think' tool allows agents to reflect before responding, especially useful
for complex tasks:

```yaml
agents:
  root:
    name: complex_thinker
    model: claude
    description: "An agent enhanced with thinking capabilities."
    instruction: |
      Before making decisions, verify all information using the think tool.
    think: true
```

Use the 'think' tool when:

- Processing complex sequential tasks
- Ensuring compliance with specific policies
- Verifying gathered information before acting

## Complete Example: Multi-Agent System

Here's a complete example combining multiple concepts:

```yaml
agents:
  root:
    name: gordon
    model: openai
    description: |
      You are a helpful assistant.
    instruction: |
      You are an expert in Docker.
      # Detailed instructions...
    sub_agents:
      - containerize
      - optimize_dockerfile

  containerize:
    name: containerize
    model: openai
    description: |
      You are a helpful assistant for containerizing applications.
    instruction: |
      # Containerization instructions...

  optimize_dockerfile:
    name: optimize_dockerfile
    model: openai
    description: |
      You are a helpful assistant for optimizing Dockerfiles.
    instruction: |
      # Optimization instructions...

models:
  openai:
    type: openai
    model: gpt-4o
    temperature: 0.7
```

## Common Configuration Patterns

### Task-Specific Workflows

Define clear workflows in the instruction section:

```yaml
instruction: |
  <TASK>
    # **Workflow:**
    # 1. **Understand the question asked by the user**
    # 2. **Use conversation context/state and tools**
    # 3. **Provide accurate information**
  </TASK>
```

### Tool Integration

Define tools that the agent can access:

```yaml
instruction: |
  **Tools:**
  You have access to the following tools:
  * `read_file(file_path: str) -> str`: Reads file content
  * `write_file(file_path: str, content: str) -> None`: Writes to files
```

### Setting Constraints

Specify behavioral constraints:

```yaml
instruction: |
  **Constraints:**
  * Use markdown for formatting
  * Be concise and avoid unnecessary verbosity
  * Follow best practices for [specific domain]
```

## Troubleshooting

- **Agent not responding correctly**: Check the instruction field for clarity
  and specificity
- **Tool errors**: Verify that tool paths and arguments are correct
- **Sub-agent communication issues**: Ensure sub-agents are properly defined and
  referenced

---

For more detailed information about agent configuration options, refer to the
[Reference Documentation](./reference.md).

For conceptual understanding of how agents work, see the
[Explanation](./explanation.md) section.
