# cagent Documentation

Welcome to the cagent documentation! This directory contains comprehensive
guides for using and understanding cagent, a powerful multi-agent AI system.

## 📚 Documentation Structure

### [User Guide](./user-guide.md)

**Start here if you're new to cagent**

Learn how to:

- Install and set up cagent
- Create your first agent
- Configure multi-agent teams
- Use built-in and external tools
- Run interactive sessions
- Best practices and troubleshooting

### [Architecture Guide](./architecture.md)

**For developers and advanced users**

Understand:

- System architecture and component interactions
- Data flow and message processing
- Extension points for custom functionality
- Performance and security considerations
- Design principles and patterns

## 🚀 Quick Start

1. **Install cagent:**

   ```bash
   git clone https://github.com/rumpl/cagent.git
   cd cagent
   task build
   ```

2. **Set up your API key:**

   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```

3. **Run your first agent:**
   ```bash
   ./cagent run examples/config/agent.yaml
   ```

## 🎯 Key Features

- **🏗️ Multi-agent architecture** - Create specialized agents for different
  domains
- **🔧 Rich tool ecosystem** - Agents can use external tools via MCP protocol
- **🔄 Smart delegation** - Agents automatically route tasks to specialists
- **📝 YAML configuration** - Simple, declarative setup
- **💭 Advanced reasoning** - Built-in tools for complex problem solving
- **🌐 Multiple AI providers** - Support for OpenAI, Anthropic, and DMR

## 🛠️ Configuration Examples

### Simple Assistant

```yaml
agents:
  root:
    name: assistant
    model: gpt4
    description: A helpful AI assistant
    instruction: |
      You are a helpful assistant. Answer questions
      accurately and concisely.

models:
  gpt4:
    type: openai
    model: gpt-4o
```

### Development Team

```yaml
agents:
  root:
    name: tech_lead
    model: claude
    description: Technical lead coordinating development
    sub_agents: [developer, reviewer]

  developer:
    name: coder
    model: claude
    description: Expert software developer
    toolsets:
      - type: mcp
        command: rust-mcp-filesystem
      - type: shell
    think: true

  reviewer:
    name: code_reviewer
    model: gpt4
    description: Code review specialist

models:
  gpt4:
    type: openai
    model: gpt-4o
  claude:
    type: anthropic
    model: claude-sonnet-4-0
```

## 📖 Additional Resources

- **[Examples](../examples/config/)** - Ready-to-use configurations
- **[Project README](../README.md)** - Project overview and quick start
- **[API Reference](https://pkg.go.dev/github.com/docker/cagent)** - Go package
  documentation
