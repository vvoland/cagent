# 🤖 cagent

A powerful, customizable multi-agent system that orchestrates AI agents with specialized capabilities and tools.

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## ✨ What is cagent?

cagent enables you to create intelligent agent teams where each agent has specialized knowledge, tools, and capabilities. Think of it as building a virtual team of experts that can collaborate to solve complex problems.

### 🎯 Key Features

- **🏗️ Multi-agent architecture** - Create specialized agents for different domains
- **🔧 Rich tool ecosystem** - Agents can use external tools and APIs via MCP protocol  
- **🔄 Smart delegation** - Agents automatically route tasks to the most suitable specialist
- **📝 YAML configuration** - Simple, declarative agent and model setup
- **💭 Advanced reasoning** - Built-in "think" tool for complex problem solving
- **🌐 Multiple AI providers** - Support for OpenAI, Anthropic, and more

## 🚀 Quick Start

### Prerequisites
- Go 1.24 or higher
- API key for your chosen AI provider (OpenAI, Anthropic, etc.)

### Installation & Setup

```bash
# Clone and build
git clone https://github.com/rumpl/cagent.git
cd cagent
go build -o cagent .

# Set your API key
export OPENAI_API_KEY=your_api_key_here
# or for Anthropic
export ANTHROPIC_API_KEY=your_api_key_here

# Run with a sample configuration
./cagent -config examples/agent.yaml
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
    type: openai
    model: gpt-4o
    temperature: 0.7
    max_tokens: 2000
```

Run it:
```bash
./cagent -config my-agent.yaml
```

## 📚 Examples & Use Cases

### Multi-Agent Development Team
```yaml
agents:
  root:
    name: dev_lead
    model: gpt4
    description: Development team lead
    instruction: |
      You coordinate a development team. Delegate coding tasks to the programmer
      and code reviews to the reviewer. Always ensure quality and best practices.
    sub_agents: [programmer, reviewer]

  programmer:
    name: programmer  
    model: gpt4
    description: Senior software engineer
    instruction: |
      You write clean, efficient code following best practices.
      Always include proper error handling and documentation.

  reviewer:
    name: reviewer
    model: claude
    description: Code review specialist  
    instruction: |
      You perform thorough code reviews, checking for bugs, security issues,
      performance problems, and adherence to coding standards.

models:
  gpt4:
    type: openai
    model: gpt-4o
    temperature: 0.3
  claude:
    type: anthropic  
    model: claude-3-5-sonnet-latest
    temperature: 0.2
```

### Research Assistant with Tools
```yaml
agents:
  root:
    name: researcher
    model: claude
    description: AI research assistant
    instruction: |
      You are a research assistant that helps users find and analyze information.
      Use web search when you need current information, and always cite your sources.
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
    think: true

models:
  claude:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.4
    max_tokens: 4000
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

## 🛠️ Command Line Usage

```bash
# Basic usage
./cagent                              # Use default config
./cagent -config my-config.yaml       # Use specific config
./cagent -agent specialist            # Start with specific agent
./cagent -prompt "Hello world"        # Start with initial prompt

# Web interface  
./cagent -web                         # Launch web UI
./cagent -web -port 8080              # Custom port

# Examples
./cagent -config examples/research.yaml -prompt "Research climate change impacts"
```

## 🔧 Configuration Reference

### Agent Configuration
```yaml
agents:
  agent_name:
    name: string              # Agent identifier
    model: string             # Model reference
    description: string       # Agent purpose  
    instruction: |            # Detailed behavior instructions
      Multi-line instructions here
    tools: []                 # Available tools (optional)
    sub_agents: []            # Sub-agent names (optional) 
    think: boolean            # Enable think tool (optional)
    add_date: boolean         # Add current date to context (optional)
```

### Model Configuration  
```yaml
models:
  model_name:
    type: string              # Provider: openai, anthropic
    model: string             # Model name: gpt-4o, claude-3-5-sonnet-latest
    temperature: float        # Randomness (0.0-1.0)
    max_tokens: integer       # Response length limit
    top_p: float             # Nucleus sampling (0.0-1.0)
    frequency_penalty: float  # Repetition penalty (0.0-2.0)
    presence_penalty: float   # Topic repetition penalty (0.0-2.0)
```

### Tool Configuration
```yaml
tools:
  - type: mcp                 # Model Context Protocol
    command: string           # Command to execute
    args: []                  # Command arguments
```

## 📖 Documentation

For detailed guides and examples:

- **[Tutorial](docs/tutorial.md)** - Step-by-step agent creation guide
- **[How-to Guide](docs/howto.md)** - Practical configuration examples  
- **[Explanation](docs/explanation.md)** - Concepts and architecture
- **[Reference](docs/reference.md)** - Complete configuration options

## 🤝 Examples

Explore the `examples/` directory for ready-to-use configurations:
- `examples/agent.yaml` - Basic assistant
- `examples/research.yaml` - Research agent with web search
- `examples/code.yaml` - Software development team
- `examples/finance.yaml` - Financial analysis specialist

## 🚀 Getting Started

1. Check out the [Tutorial](docs/tutorial.md) for your first agent
2. Browse [Examples](examples/) for inspiration  
3. Read the [How-to Guide](docs/howto.md) for advanced patterns
4. Consult the [Reference](docs/reference.md) for all options

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.
