# User Guide

This guide will help you get started with cagent and learn how to use its
powerful multi-agent system to accomplish various tasks.

## What is cagent?

cagent is a powerful, customizable multi-agent system that orchestrates AI
agents with specialized capabilities and tools. It enables you to create
intelligent agent teams where each agent has specialized knowledge, tools, and
capabilities.

## Quick Start

### Prerequisites

- Go 1.24 or higher
- API key for your chosen AI provider (OpenAI, Anthropic, etc.)

### Installation

1. **Clone and build the project:**

   ```bash
   git clone https://github.com/rumpl/cagent.git
   cd cagent
   task build
   ```

2. **Set your API key:**

   ```bash
   # For OpenAI
   export OPENAI_API_KEY=your_api_key_here

   # For Anthropic
   export ANTHROPIC_API_KEY=your_api_key_here
   ```

3. **Run your first agent:**
   ```bash
   ./cagent run examples/config/agent.yaml
   ```

## Core Concepts

### Agent Hierarchy

cagent organizes agents in a hierarchical structure:

- **Root Agent**: The main entry point that coordinates the system
- **Sub-Agents**: Specialized agents for specific domains or tasks
- **Tools**: External capabilities via Model Context Protocol (MCP)
- **Models**: AI providers and their configurations

### Task Delegation

The system uses intelligent task delegation:

1. User interacts with the root agent
2. Root agent analyzes the request
3. If specialized knowledge is needed, delegates to appropriate sub-agent
4. Sub-agent processes the task using its tools and expertise
5. Results flow back to the root agent and user

## Configuration

cagent uses YAML configuration files to define agents, models, and tools.

### Basic Agent Configuration

```yaml
agents:
  root:
    name: assistant
    model: gpt4
    description: A helpful AI assistant
    instruction: |
      You are a knowledgeable assistant that helps users with various tasks.
      Be helpful, accurate, and concise in your responses.

models:
  gpt4:
    type: openai
    model: gpt-4o
```

### Multi-Agent Configuration

```yaml
agents:
  root:
    name: manager
    model: gpt4
    description: Project manager that delegates tasks
    instruction: |
      You are a project manager that coordinates different specialists.
      Delegate tasks to the appropriate team members.
    sub_agents:
      - developer
      - designer

  developer:
    name: coder
    model: claude
    description: Expert software developer
    instruction: |
      You are an expert developer. Focus on coding tasks,
      code review, and technical implementation.
    toolsets:
      - type: filesystem

  designer:
    name: designer
    model: gpt4
    description: UI/UX design specialist
    instruction: |
      You are a UI/UX designer. Focus on design tasks,
      user experience, and visual elements.

models:
  gpt4:
    type: openai
    model: gpt-4o

  claude:
    type: anthropic
    model: claude-sonnet-4-0
    max_tokens: 64000
```

### Configuration Reference

#### Agent Properties

| Property      | Type    | Description                     | Required |
| ------------- | ------- | ------------------------------- | -------- |
| `name`        | string  | Agent identifier                | ✓        |
| `model`       | string  | Model reference                 | ✓        |
| `description` | string  | Agent purpose                   | ✓        |
| `instruction` | string  | Detailed behavior instructions  | ✓        |
| `sub_agents`  | array   | List of sub-agent names         | ✗        |
| `toolsets`    | array   | Available tools                 | ✗        |
| `think`       | boolean | Enable think tool               | ✗        |
| `todo`        | boolean | Enable todo list tool           | ✗        |
| `memory.path` | string  | SQLite database path for memory | ✗        |
| `add_date`    | boolean | Add current date to context     | ✗        |

#### Model Properties

| Property            | Type    | Description                                      | Required |
| ------------------- | ------- | ------------------------------------------------ | -------- |
| `type`              | string  | Provider: `openai`, `anthropic`, `dmr`           | ✓        |
| `model`             | string  | Model name (e.g., `gpt-4o`, `claude-sonnet-4-0`) | ✓        |
| `temperature`       | float   | Randomness (0.0-1.0)                             | ✗        |
| `max_tokens`        | integer | Response length limit                            | ✗        |
| `top_p`             | float   | Nucleus sampling (0.0-1.0)                       | ✗        |
| `frequency_penalty` | float   | Repetition penalty (0.0-2.0)                     | ✗        |
| `presence_penalty`  | float   | Topic repetition penalty (0.0-2.0)               | ✗        |
| `base_url`          | string  | Custom API endpoint                              | ✗        |

#### Tool Configuration

**Local (stdio) MCP Server**

```yaml
toolsets:
  - type: mcp # Model Context Protocol
    command: string # Command to execute
    args: [] # Command arguments
    tools: [] # Optional: List of specific tools to enable
    env: [] # Environment variables for this tool
    env_file: [] # Environment variable files
```

**Remote (sse or streamable) MCP Server**

```yaml
toolsets:
  - type: mcp # Model Context Protocol
    remote:
      url: string # Base URL to connect to
      transport_type: string # Type of MCP transport (sse or streamable)
      headers:
        key: value # HTTP headers. Mainly used for auth
    tools: [] # Optional: List of specific tools to enable
```

**Builtin tools:**

```yaml
toolsets:
  - type: filesystem # Access to local files
  - type: shell # Shell access
```

## Running Agents

### Command Line Interface

cagent provides several commands:

```bash
# Run an agent interactively
$ cagent run config.yaml

# Run a specific agent from the config
$ cagent run config.yaml -a agent_name

# Enable debug logging
$ cagent run config.yaml --debug

# Start web interface
$ cagent web -d ./folder/with/agent_files /tmp/session.db

# Start UI interface
$ cagent ui config.yaml

# Start API server
$ cagent api config.yaml

# Initialize a new project
$ cagent init

# Run evaluations
$ cagent eval config.yaml
```

### Interactive Commands

During an interactive session, you can use special commands:

| Command  | Description                              |
| -------- | ---------------------------------------- |
| `/exit`  | Exit the program                         |
| `/reset` | Clear conversation history               |
| `/eval`  | Save current conversation for evaluation |

## Built-in Tools

cagent includes several built-in tools that agents can use:

### Think Tool

The think tool allows agents to reason through problems step by step:

```yaml
agents:
  root:
    # ... other config
    think: true
```

### Todo Tool

The todo tool helps agents manage task lists:

```yaml
agents:
  root:
    # ... other config
    todo: true
```

### Memory Tool

The memory tool provides persistent storage:

```yaml
agents:
  root:
    # ... other config
    memory:
      path: "./agent_memory.db"
```

### Task Transfer Tool

All agents automatically have access to the task transfer tool, which allows
them to delegate tasks to other agents:

```
transfer_task(agent="developer", task="Create a login form", expected_output="HTML and CSS code")
```

## External Tools (MCP)

cagent supports external tools through the Model Context Protocol (MCP). This
allows agents to use a wide variety of external capabilities.

### Available MCP Tools

Common MCP tools include:

- **Filesystem**: Read/write files
- **Shell**: Execute shell commands
- **Database**: Query databases
- **Web**: Make HTTP requests
- **Git**: Version control operations

### Installing MCP Tools

Example installation of filesystem tool:

```bash
# Install Rust-based MCP filesystem tool
cargo install rust-mcp-filesystem
```

a
Then configure in your agent:

```yaml
toolsets:
  - type: mcp
    command: rust-mcp-filesystem
    args: ["--allow-write", "."]
```

## Examples

### Development Team

A complete development team with specialized roles:

```yaml
agents:
  root:
    name: tech_lead
    model: claude
    description: Technical lead coordinating development
    instruction: |
      You are a technical lead managing a development team.
      Coordinate tasks between developers and ensure quality.
    sub_agents: [developer, reviewer, tester]

  developer:
    name: coder
    model: claude
    description: Expert software developer
    instruction: |
      You are an expert developer. Write clean, efficient code
      and follow best practices.
    toolsets:
      - type: filesystem
      - type: shell
    think: true

  reviewer:
    name: code_reviewer
    model: gpt4
    description: Code review specialist
    instruction: |
      You are a code review expert. Focus on code quality,
      security, and maintainability.
    toolsets:
      - type: filesystem

  tester:
    name: qa_engineer
    model: gpt4
    description: Quality assurance engineer
    instruction: |
      You are a QA engineer. Write tests and ensure
      software quality.
    toolsets:
      - type: shell
    todo: true

models:
  gpt4:
    type: openai
    model: gpt-4o

  claude:
    type: anthropic
    model: claude-sonnet-4-0
    max_tokens: 64000
```

### Research Assistant

A research-focused agent with web access:

```yaml
agents:
  root:
    name: researcher
    model: claude
    description: Research assistant with web access
    instruction: |
      You are a research assistant. Help users find information,
      analyze data, and provide insights.
    toolsets:
      - type: mcp
        command: mcp-web-search
        args: ["--provider", "duckduckgo"]
    think: true
    memory:
      path: "./research_memory.db"

models:
  claude:
    type: anthropic
    model: claude-sonnet-4-0
    max_tokens: 64000
```

## Best Practices

### Agent Design

1. **Single Responsibility**: Each agent should have a clear, focused purpose
2. **Clear Instructions**: Provide detailed, specific instructions for each
   agent
3. **Appropriate Tools**: Give agents only the tools they need
4. **Hierarchy**: Use sub-agents for specialized tasks

### Configuration Management

1. **Validation**: Always validate your configuration before running
2. **Environment Variables**: Use environment variables for sensitive data
3. **Modularity**: Break complex configurations into smaller, reusable pieces
4. **Documentation**: Document your agent configurations

### Tool Usage

1. **Minimal Permissions**: Give tools only necessary permissions
2. **Error Handling**: Consider how agents will handle tool failures
3. **Security**: Be cautious with shell access and file system permissions
4. **Testing**: Test tool combinations thoroughly

## Troubleshooting

### Common Issues

**Agent not responding:**

- Check API keys are set correctly
- Verify model name matches provider
- Check network connectivity

**Tool errors:**

- Ensure MCP tools are installed
- Check file permissions
- Verify tool arguments

**Configuration errors:**

- Validate YAML syntax
- Check all referenced agents exist
- Ensure all models are defined

### Debug Mode

Enable debug logging for detailed information:

```bash
./cagent run config.yaml --debug
```

### Log Analysis

Check logs for:

- API call errors
- Tool execution failures
- Configuration validation issues
- Network connectivity problems

## Advanced Usage

### Custom Memory

Implement custom memory strategies:

```yaml
memory:
  path: "./custom_memory.db"
  # Memory will persist across sessions
```

### Multi-Model Teams

Use different models for different capabilities:

```yaml
agents:
  analyst:
    model: claude_reasoning # Good for analysis
  creator:
    model: gpt4_creative # Good for creative tasks
  coder:
    model: claude_coding # Good for coding

models:
  claude_reasoning:
    type: anthropic
    model: claude-sonnet-4-0
    temperature: 0.2

  gpt4_creative:
    type: openai
    model: gpt-4o
    temperature: 0.8

  claude_coding:
    type: anthropic
    model: claude-sonnet-4-0
    temperature: 0.1
```

### Integration with CI/CD

Use cagent in automated workflows:

```bash
# Example: Code review automation
./cagent run review_agent.yaml < changed_files.txt
```

This guide should help you get started with cagent and build powerful
multi-agent systems. For more advanced topics, see the Architecture Guide and
API Reference.
