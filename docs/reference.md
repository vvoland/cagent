# Agent Configuration Reference

This reference documentation provides detailed information about all configuration options available when creating
agents using YAML. Use this guide as a comprehensive reference for all available fields, options, and their expected
values.

## YAML Configuration Structure

The agent configuration YAML file consists of three main sections:

```yaml
agents:
  # Agent definitions

models:
  # Model configurations
```

## Agents Section

The `agents` section defines all agents in the system, including the root agent and any sub-agents.

### Root Agent Configuration

| Field         | Type    | Required | Description                                                       |
| ------------- | ------- | -------- | ----------------------------------------------------------------- |
| `name`        | String  | Yes      | Unique identifier for the agent                                   |
| `model`       | String  | Yes      | Reference to a model defined in the models section                |
| `description` | String  | Yes      | Brief description of the agent's purpose                          |
| `instruction` | String  | Yes      | Detailed instructions guiding the agent's behavior                |
| `tools`       | Array   | No       | List of tools available to the agent                              |
| `sub_agents`  | Array   | No       | List of sub-agent names that can be called by this agent          |
| `think`       | Boolean | No       | When true, enables the agent to use the think tool for reflection |
| `add_date`    | Boolean | No       | When true, adds the current date to the agent's context           |

Example:

```yaml
agents:
  root:
    name: research_assistant
    model: openai
    description: A research assistant that provides well-sourced information
    instruction: |
      You are a research assistant that provides accurate, well-sourced information.
      Always cite your sources and prefer recent information.
    tools:
      - type: search
    sub_agents:
      - fact_checker
    think: true
```

### Sub-Agent Configuration

Sub-agents have the same configuration options as the root agent but are defined as separate entities under the `agents`
section and referenced in the `sub_agents` array of the parent agent.

Example:

```yaml
agents:
  root:
    # Root agent config
    sub_agents:
      - fact_checker

  fact_checker:
    name: fact_checker
    model: claude
    description: Verifies factual accuracy of information
    instruction: |
      You are specialized in verifying the factual accuracy of information.
      Check for inconsistencies and validate against reliable sources.
```

## Models Section

The `models` section defines the language models that agents can use.

### Model Configuration

| Field               | Type    | Required | Description                                                            |
| ------------------- | ------- | -------- | ---------------------------------------------------------------------- |
| `type`              | String  | Yes      | The provider of the model (e.g., `openai`, `anthropic`)                |
| `model`             | String  | Yes      | The specific model to use (e.g., `gpt-4o`, `claude-3-5-sonnet-latest`) |
| `temperature`       | Float   | No       | Controls randomness in output (0.0-1.0)                                |
| `max_tokens`        | Integer | No       | Maximum number of tokens in the response                               |
| `top_p`             | Float   | No       | Nucleus sampling parameter (0.0-1.0)                                   |
| `frequency_penalty` | Float   | No       | Penalty for new token based on frequency (0.0-2.0)                     |
| `presence_penalty`  | Float   | No       | Penalty for new token based on presence (0.0-2.0)                      |

Example:

```yaml
models:
  openai:
    type: openai
    model: gpt-4o
    temperature: 0.7
    max_tokens: 1500
    top_p: 1.0

  claude:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.5
```

## Tools Section

The `tools` section defines global tools that can be referenced by agents. Tools can also be defined directly within an
agent's configuration.

### Tool Configuration

| Field     | Type   | Required            | Description                                        |
| --------- | ------ | ------------------- | -------------------------------------------------- |
| `type`    | String | Yes                 | The type of tool (e.g., `mcp`, `search`, `custom`) |
| `command` | String | For some tool types | The command to execute (for `mcp` type tools)      |
| `args`    | Array  | For some tool types | Arguments to pass to the command                   |

Example of global tool definition:

```yaml
tools:
  airbnb:
    type: mcp
    command: npx
    args: ["-y", "@openbnb/mcp-server-airbnb", "--ignore-robots-txt"]
```

Example of tool within an agent:

```yaml
agents:
  root:
    # Basic configuration
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

### Common Tool Types

#### MCP Tools

Multi-Command Protocol (MCP) tools allow agents to execute external commands and receive their output.

```yaml
tools:
  - type: mcp
    command: npx
    args: ["-y", "@openbnb/mcp-server-airbnb", "--ignore-robots-txt"]
```

#### File System Tools

These tools enable agents to interact with the file system:

- `read_file(file_path: str) -> str`: Reads file content
- `write_file(file_path: str, content: str) -> None`: Writes to files
- `list_files_in_directory(directory_path: str) -> list`: Lists files in a directory
- `search_files(path: str, pattern: str) -> list`: Searches for files matching a pattern

#### Think Tool

Enables an agent to reflect on complex problems before responding. Activated by setting `think: true` in the agent
configuration.

## Instruction Field Best Practices

The `instruction` field is crucial for defining agent behavior. It typically includes sections like:

### Task Definition

```yaml
instruction: |
  <TASK>
    # **Workflow:**
    # 1. First step
    # 2. Second step
    # ...
  </TASK>
```

### Tools Description

```yaml
instruction: |
  **Tools:**
  You have access to the following tools:
  * `tool_name(param: type) -> return_type`: Description
  * ...
```

### Constraints

```yaml
instruction: |
  **Constraints:**
  * Constraint 1
  * Constraint 2
  * ...
```

## Complete Reference Example

```yaml
agents:
  root:
    name: research_assistant
    model: openai
    description: A research assistant that provides well-sourced information
    instruction: |
      You are a research assistant that provides accurate, well-sourced information.
      Always cite your sources and prefer recent information.

      <TASK>
        # **Workflow:**
        # 1. Understand the user's question
        # 2. Search for relevant information
        # 3. Analyze and verify the information
        # 4. Provide a comprehensive answer with sources
      </TASK>

      **Tools:**
      You have access to the following tools:
      * `search(query: str) -> str`: Searches the web for information

      **Constraints:**
      * Use markdown for formatting
      * Always cite sources
      * Be concise but thorough
    tools:
      - type: mcp
        command: search
        args: []
    sub_agents:
      - fact_checker
    think: true

  fact_checker:
    name: fact_checker
    model: claude
    description: Verifies factual accuracy of information
    instruction: |
      You are specialized in verifying the factual accuracy of information.
      Check for inconsistencies and validate against reliable sources.

models:
  openai:
    type: openai
    model: gpt-4o
    temperature: 0.7
    max_tokens: 1500

  claude:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.5
```
