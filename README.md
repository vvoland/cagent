# cagent

A customizable multi-agent system.

## Overview

cagent you to create and manage multiple AI agents with specialized
capabilities. Each agent can be configured with different instructions, tools,
and model parameters. Agents can work together, with the ability to transfer
conversations between them when specialized knowledge is needed.

## Features

- **Multi-agent architecture**: Define specialized agents for different tasks
- **Tool-based interaction**: Agents can use tools to interact with the system
- **Hierarchical delegation**: Root agents can transfer control to sub-agents
- **Configurable via YAML**: Easy configuration of agents and models
- **Interactive console**: Conversational interface with the agents

## Getting Started

### Prerequisites

- Go 1.24 or higher
- OpenAI API key

### Installation

```bash
# Clone the repository
git clone https://github.com/rumpl/cagent.git
cd cagent

# Build the application
go build -o cagent .
```

### Environment Setup

Set your OpenAI API key as an environment variable:

```bash
export OPENAI_API_KEY=your_api_key_here
```

### Configuration

Create an `agent.yaml` file to define your agents and their capabilities. See
the examples directory for sample configurations.

### Usage

```bash
# Run with default configuration
./cagent

# Run with a specific configuration file
./cagent -config my_agents.yaml

# Run a specific agent
./cagent -agent coding_assistant

# Start with an initial prompt
./cagent -prompt "Create a Python script to analyze data from CSV files"
```

## Interactive Mode

When you run cagent, it starts in interactive mode where you can chat with the
agent:

```
You: How can you help me?
[Agent response will appear here]

You: Create a simple web server in Go
[Agent response will appear here]
```

Type `exit` to end the session.

## Configuration Format

The configuration file uses YAML format to define:

- Agent definitions (instructions, available tools, sub-agents)
- Model settings (type, parameters, etc.)

See the examples directory for sample configurations.
