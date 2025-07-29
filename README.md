# 🤖 cagent

A powerful, customizable multi-agent system that orchestrates AI agents with
specialized capabilities and tools.

## ✨ What is cagent?

cagent enables you to create intelligent agent teams where each agent has
specialized knowledge, tools, and capabilities. Think of it as building a
virtual team of experts that can collaborate to solve complex problems.

### 🎯 Key Features

- **🏗️ Multi-agent architecture** - Create specialized agents for different
  domains
- **🔧 Rich tool ecosystem** - Agents can use external tools and APIs via MCP
  protocol
- **🔄 Smart delegation** - Agents automatically route tasks to the most
  suitable specialist
- **📝 YAML configuration** - Simple, declarative agent and model setup
- **💭 Advanced reasoning** - Built-in "think", "todo" and "memory" tools for
  complex problem solving
- **🌐 Multiple AI providers** - Support for OpenAI, Anthropic and DMR

## 🚀 Quick Start

### Prerequisites

- Go 1.24 or higher
- API key for your chosen AI provider (OpenAI, Anthropic, etc.)

### Installation & Setup

```bash
# Clone and build
git clone https://github.com/docker/cagent.git
cd cagent
task build

# Set your API key
export OPENAI_API_KEY=your_api_key_here
# or for Anthropic
export ANTHROPIC_API_KEY=your_api_key_here

# Run with a sample configuration
./bin/cagent run examples/config/code.yaml
# or specify a different agent from the config
./bin/cagent run examples/config/code.yaml -a root
```

### Your First Agent

Create `my-agent.yaml`:

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
    provider: openai
    model: gpt-4o
```

Run it:

```bash
./bin/cagent run my-agent.yaml
# or specify a different agent from the config
./bin/cagent run my-agent.yaml -a root
```

## 🎯 Core Concepts

### Agent Hierarchy

- **Root Agent**: Main entry point that coordinates the system
- **Sub-Agents**: Specialized agents for specific domains or tasks
- **Tools**: External capabilities via Model Context Protocol (MCP)
- **Models**: AI providers and their configurations

### Delegation Flow

1. User interacts with root agent
2. Root agent analyzes the request
3. Delegates to appropriate sub-agent if specialized knowledge needed
4. Sub-agent processes task using its tools and expertise
5. Results flow back to root agent and user

## 🔧 Configuration Reference

### Agent Configuration

```yaml
agents:
  agent_name:
    name: string # Agent identifier
    model: string # Model reference
    description: string # Agent purpose
    instruction: string # Detailed behavior instructions
    tools: [] # Available tools (optional)
    sub_agents: [] # Sub-agent names (optional)
    add_date: boolean # Add current date to context (optional)
```

### Model Configuration

```yaml
models:
  model_name:
    provider: string # Provider: openai, anthropic, dmr
    model: string # Model name: gpt-4o, claude-3-5-sonnet-latest
    temperature: float # Randomness (0.0-1.0)
    max_tokens: integer # Response length limit
    top_p: float # Nucleus sampling (0.0-1.0)
    frequency_penalty: float # Repetition penalty (0.0-2.0)
    presence_penalty: float # Topic repetition penalty (0.0-2.0)
    parallel_tool_calls: boolean
```

#### Model Examples

```yaml

#OpenAI API
models:
  openai:
    provider: openai
    model: gpt-4o

#Anthropic API
models:
  claude:
    provider: anthropic
    model: claude-sonnet-4-0

#Docker Model Runner
models:
  qwen:
    provider: dmr
    model: ai/qwen3

#Ollama
models:
  ollama:
    provider: openai
    model: llama3
    base_url: http://localhost:11434/v1

```

### Tool Configuration

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
    - type: filesystem
    - type: think # Enable think tool (optional)
    - type: todo # Enable the todo list tool (optional)
      shared: boolean # Should the todo list be shared (optional)
    - type: memory 
      path: # Path to the sqlite database for memory storage (optional)
```

## 🤝 Examples

Explore the [examples/config](examples/config/) directory for ready-to-use configurations:

- [examples/config/agent.yaml](examples/config/agent.yaml) - Basic assistant
- [examples/config/code.yaml](examples/config/code.yaml) - Software development team
- [examples/config/finance.yaml](examples/config/finance.yaml) - Financial analysis specialist
