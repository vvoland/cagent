# User Guide

This guide will help you get started with cagent and learn how to use its
powerful multi-agent system to accomplish various tasks.

## What is cagent?

cagent is a powerful, customizable multi-agent system that orchestrates AI
agents with specialized capabilities and tools. It features:

- **🏗️ Multi-tenant architecture** with client isolation and session management
- **🔧 Rich tool ecosystem** via Model Context Protocol (MCP) integration
- **🤖 Hierarchical agent system** with intelligent task delegation
- **🌐 Multiple interfaces** including CLI, Web UI, TUI, API server, and MCP server
- **📦 Agent distribution** via Docker registry integration
- **🔒 Security-first design** with proper client scoping and resource isolation
- **⚡ Event-driven streaming** for real-time interactions
- **🧠 Multi-model support** (OpenAI, Anthropic, DMR, Docker AI Gateway)

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

   # For Docker AI Gateway (if using Docker Desktop)
   # Authentication is handled automatically via Docker Desktop
   ```

3. **Run your first agent:**

   ```bash
   # Interactive CLI mode
   ./bin/cagent run examples/config/agent.yaml

   # Or start the web interface
   ./bin/cagent web -d ./examples/config /tmp/session.db

   # Or start as MCP server for external clients
   ./bin/cagent mcp server --agents-dir ./examples/config --port 8080
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
    model: gpt4
    description: A helpful AI assistant
    instruction: |
      You are a knowledgeable assistant that helps users with various tasks.
      Be helpful, accurate, and concise in your responses.

models:
  gpt4:
    provider: openai
    model: gpt-4o
```

### Multi-Agent Configuration

```yaml
agents:
  root:
    model: gpt4
    description: Project manager that delegates tasks
    instruction: |
      You are a project manager that coordinates different specialists.
      Delegate tasks to the appropriate team members.
    sub_agents:
      - developer
      - designer

  developer:
    model: claude
    description: Expert software developer
    instruction: |
      You are an expert developer. Focus on coding tasks,
      code review, and technical implementation.
    toolsets:
      - type: filesystem

  designer:
    model: gpt4
    description: UI/UX design specialist
    instruction: |
      You are a UI/UX designer. Focus on design tasks,
      user experience, and visual elements.

models:
  gpt4:
    provider: openai
    model: gpt-4o

  claude:
    provider: anthropic
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

cagent provides multiple interfaces and deployment modes:

```bash
# Interactive CLI mode
$ ./bin/cagent run config.yaml
$ ./bin/cagent run config.yaml -a agent_name  # Run specific agent
$ ./bin/cagent run config.yaml --debug        # Enable debug logging

# Web Interface (recommended for multi-session use)
$ ./bin/cagent web -d ./config_directory /tmp/session.db
$ ./bin/cagent web -d ./config_directory /tmp/session.db --port 3000

# Terminal UI
$ ./bin/cagent tui config.yaml

# MCP Server Mode (for external clients like Claude Code)
$ ./bin/cagent mcp server --agents-dir ./config_directory
$ ./bin/cagent mcp server --port 8080 --path /mcp --agents-dir ./config

# API Server (HTTP REST API)
$ ./bin/cagent api config.yaml
$ ./bin/cagent api config.yaml --port 8080

# Docker AI Gateway Integration
$ ./bin/cagent run config.yaml --gateway https://api.docker.com

# Project Management
$ ./bin/cagent init                          # Initialize new project
$ ./bin/cagent eval config.yaml             # Run evaluations
$ ./bin/cagent pull docker.io/user/agent    # Pull agent from registry
$ ./bin/cagent push docker.io/user/agent    # Push agent to registry
```

### Interface-Specific Features

#### CLI Interactive Commands

During CLI sessions, you can use special commands:

| Command  | Description                              |
| -------- | ---------------------------------------- |
| `/exit`  | Exit the program                         |
| `/reset` | Clear conversation history               |
| `/eval`  | Save current conversation for evaluation |

#### Web Interface Features

- **Multi-session management**: Create and switch between multiple agent sessions
- **Session persistence**: Conversations saved to SQLite database
- **Real-time streaming**: Live updates as agents process requests
- **Agent switching**: Easy switching between different agents in the same session
- **History management**: Full conversation history with search and export

#### MCP Server Mode

- **External client integration**: Works with Claude Code, Cursor, and other MCP clients
- **Session isolation**: Each MCP client gets isolated sessions
- **Tool exposure**: Agents accessible as MCP tools for external use
- **Real-time streaming**: SSE-based streaming responses
- **Multi-client support**: Handle multiple concurrent MCP clients

## Built-in Tools

cagent includes several built-in tools that agents can use:

### Think Tool

The think tool allows agents to reason through problems step by step:

```yaml
agents:
  root:
    # ... other config
    toolsets:
      - type: think
```

### Todo Tool

The todo tool helps agents manage task lists:

```yaml
agents:
  root:
    # ... other config
    toolsets:
      - type: todo
```

### Memory Tool

The memory tool provides persistent storage:

```yaml
agents:
  root:
    # ... other config
    toolsets:
      - type: memory
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
- **Browser**: Web browsing and automation
- **Code**: Programming language specific tools
- **API**: REST API integration tools

### Installing MCP Tools

Example installation of filesystem tool:

```bash
# Install Rust-based MCP filesystem tool
cargo install rust-mcp-filesystem

# Install other popular MCP tools
npm install -g @modelcontextprotocol/server-filesystem
npm install -g @modelcontextprotocol/server-git
npm install -g @modelcontextprotocol/server-web
```

### Configuring MCP Tools

**Local (stdio) MCP Server:**

```yaml
toolsets:
  - type: mcp
    command: rust-mcp-filesystem
    args: ["--allow-write", "."]
    tools: ["read_file", "write_file"] # Optional: specific tools only
    env:
      - "RUST_LOG=debug"
```

**Remote (SSE) MCP Server:**

```yaml
toolsets:
  - type: mcp
    remote:
      url: "https://mcp-server.example.com"
      transport_type: "sse"
      headers:
        Authorization: "Bearer your-token-here"
    tools: ["search_web", "fetch_url"]
```

## Examples

### Development Team

A complete development team with specialized roles:

```yaml
agents:
  root:
    model: claude
    description: Technical lead coordinating development
    instruction: |
      You are a technical lead managing a development team.
      Coordinate tasks between developers and ensure quality.
    sub_agents: [developer, reviewer, tester]

  developer:
    model: claude
    description: Expert software developer
    instruction: |
      You are an expert developer. Write clean, efficient code
      and follow best practices.
    toolsets:
      - type: filesystem
      - type: shell
      - type: think

  reviewer:
    model: gpt4
    description: Code review specialist
    instruction: |
      You are a code review expert. Focus on code quality,
      security, and maintainability.
    toolsets:
      - type: filesystem

  tester:
    model: gpt4
    description: Quality assurance engineer
    instruction: |
      You are a QA engineer. Write tests and ensure
      software quality.
    toolsets:
      - type: shell
      - type: todo

models:
  gpt4:
    provider: openai
    model: gpt-4o

  claude:
    provider: anthropic
    model: claude-sonnet-4-0
    max_tokens: 64000
```

### Research Assistant

A research-focused agent with web access:

```yaml
agents:
  root:
    model: claude
    description: Research assistant with web access
    instruction: |
      You are a research assistant. Help users find information,
      analyze data, and provide insights.
    toolsets:
      - type: mcp
        command: mcp-web-search
        args: ["--provider", "duckduckgo"]
      - type: todo
      - type: memory
        path: "./research_memory.db"

models:
  claude:
    provider: anthropic
    model: claude-sonnet-4-0
    max_tokens: 64000
```

## Advanced Features

### Agent Store and Distribution

cagent supports distributing agents via Docker registries:

```bash
# Pull an agent from a registry
./bin/cagent pull docker.io/username/my-agent:latest

# Push your agent to a registry
./bin/cagent push docker.io/username/my-agent:latest
```

**Agent References:**

- File agents: `my-agent.yaml` (relative path)
- Store agents: `docker.io/username/my-agent:latest` (full Docker reference)

### Docker AI Gateway Integration

When using Docker Desktop, cagent can integrate with Docker's AI Gateway:

```yaml
models:
  gateway_gpt4:
    provider: openai
    model: gpt-4o
    base_url: https://api.docker.com/v1
    # Authentication handled automatically via Docker Desktop
```

```bash
# Use gateway flag to override model endpoints
./bin/cagent run config.yaml --gateway https://api.docker.com
```

### Session Management

**Web Interface Sessions:**

- Persistent SQLite storage
- Multi-agent conversations in single session
- Session history and search
- Export capabilities

**MCP Server Sessions:**

- Client-isolated sessions
- Session creation and management via MCP tools
- Real-time streaming responses
- Session timeout and cleanup

## Best Practices

### Agent Design

1. **Single Responsibility**: Each agent should have a clear, focused purpose
2. **Clear Instructions**: Provide detailed, specific instructions for each agent
3. **Appropriate Tools**: Give agents only the tools they need
4. **Hierarchy Design**: Use sub-agents for specialized tasks and clear delegation paths
5. **Model Selection**: Choose appropriate models for different capabilities (reasoning vs creativity)

### Configuration Management

1. **Validation**: Always validate your configuration before running
2. **Environment Variables**: Use environment variables for sensitive data
3. **Modularity**: Break complex configurations into smaller, reusable pieces
4. **Documentation**: Document your agent configurations and tool permissions
5. **Version Control**: Track agent configurations in git for reproducibility

### Tool Usage

1. **Minimal Permissions**: Give tools only necessary permissions
2. **Error Handling**: Consider how agents will handle tool failures
3. **Security**: Be cautious with shell access and file system permissions
4. **Testing**: Test tool combinations thoroughly in isolation
5. **MCP Tool Lifecycle**: Properly handle MCP tool start/stop lifecycle

### Production Deployment

1. **MCP Server Mode**: Use MCP server for external integrations
2. **Web Interface**: Use web mode for multi-user scenarios
3. **Session Limits**: Configure appropriate session limits and timeouts
4. **Monitoring**: Enable debug logging for troubleshooting
5. **Resource Management**: Monitor memory and CPU usage for concurrent sessions
6. **Client Isolation**: Ensure proper client scoping in multi-tenant deployments

## Troubleshooting

### Common Issues

**Agent not responding:**

- Check API keys are set correctly
- Verify model name matches provider
- Check network connectivity
- Ensure Docker Desktop is running (for Docker AI Gateway)
- Verify gateway authentication (check Docker Desktop login)

**Tool errors:**

- Ensure MCP tools are installed and accessible
- Check file permissions for filesystem tools
- Verify tool arguments and command paths
- Test MCP tools independently before integration
- Check tool lifecycle (start/stop) in debug logs

**Configuration errors:**

- Validate YAML syntax
- Check all referenced agents exist
- Ensure all models are defined
- Verify toolset configurations
- Check agent hierarchy (sub_agents references)

**Session and connectivity issues:**

- Verify port availability for web/MCP server modes
- Check SQLite database permissions for web interface
- Test MCP endpoint accessibility (curl test)
- Verify client isolation in multi-tenant scenarios
- Check session timeouts and limits

**Performance issues:**

- Monitor memory usage with multiple concurrent sessions
- Check for tool resource leaks
- Verify proper session cleanup
- Monitor streaming response performance

### Debug Mode

Enable debug logging for detailed information:

```bash
# CLI mode
./bin/cagent run config.yaml --debug

# Web interface
./bin/cagent web -d ./config /tmp/session.db --debug

# MCP server
./bin/cagent mcp server --agents-dir ./config --debug
```

### Log Analysis

Check logs for:

- API call errors and rate limiting
- Tool execution failures and timeouts
- Configuration validation issues
- Network connectivity problems
- MCP protocol handshake issues
- Session creation and cleanup events
- Client isolation boundary violations
- Docker AI Gateway authentication failures

### Testing MCP Integration

Test MCP server functionality:

```bash
# Start MCP server
./bin/cagent mcp server --agents-dir ./examples/config --port 8080 --debug

# Test with curl (check server is running)
curl -N http://localhost:8080/mcp/sse

# Run MCP test client
cd examples/mcptesting
go run test-mcp-client.go
```

### Agent Store Issues

```bash
# Check agent resolution
./bin/cagent mcp server --agents-dir ./config --debug
# Look for "Agent resolved" messages in logs

# Test Docker registry connectivity
docker pull docker.io/username/agent:latest

# Verify agent content
./bin/cagent pull docker.io/username/agent:latest
```

## Integration Examples

### MCP Client Integration

Using cagent agents from external MCP clients:

```javascript
// Example: Using cagent from Claude Code or Cursor
const mcp = require("@modelcontextprotocol/client");

// Connect to cagent MCP server
const client = new mcp.Client({
  url: "http://localhost:8080/mcp/sse",
  transport: "sse",
});

// List available agents
const agents = await client.callTool("list_agents", {});

// Create a session with a specific agent
const session = await client.callTool("create_agent_session", {
  agent_spec: "developer",
  initial_message: "Help me debug this Python code",
});

// Send messages to the agent
const response = await client.callTool("send_message", {
  session_id: session.session_id,
  message:
    "def fibonacci(n): return n if n <= 1 else fibonacci(n-1) + fibonacci(n-2)",
});
```

### Custom Memory Strategies

Implement persistent memory across sessions:

```yaml
agents:
  researcher:
    model: claude
    memory:
      path: "./research_memory.db"
    instruction: |
      You are a research assistant with persistent memory.
      Remember important findings and reference previous research.
```

### Multi-Model Teams with Gateway Integration

```yaml
models:
  # Local model for fast responses
  claude_local:
    provider: anthropic
    model: claude-sonnet-4-0
    temperature: 0.2

  # Gateway model for enhanced capabilities
  gpt4_gateway:
    provider: openai
    model: gpt-4o
    base_url: https://api.docker.com/v1
    temperature: 0.1

  # Creative model for content generation
  gpt4_creative:
    provider: openai
    model: gpt-4o
    temperature: 0.8

agents:
  analyst:
    model: claude_local
    description: Fast analysis and reasoning

  coder:
    model: gpt4_gateway
    description: Enhanced coding with gateway models

  writer:
    model: gpt4_creative
    description: Creative content generation
```

This guide should help you get started with cagent and build powerful
multi-agent systems. For more advanced topics, see the Architecture Guide and
API Reference.
