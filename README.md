# cagent

A customizable multi-agent system for orchestrating AI agents with specialized capabilities.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/rumpl/cagent.git
cd cagent

# Build the application
go build -o cagent .

# Set your OpenAI API key
export OPENAI_API_KEY=your_api_key_here

# Run with default configuration
./cagent
```

## Introduction

cagent lets you create and manage multiple AI agents with specialized capabilities. Each agent can be configured with different instructions, tools, and model parameters. Agents work together through hierarchical delegation, transferring conversations when specialized knowledge is needed.

### Key Features

- **Multi-agent architecture**: Define specialized agents for different tasks
- **Tool-based interaction**: Agents can use tools to interact with the system
- **Hierarchical delegation**: Root agents can transfer control to sub-agents
- **Configurable via YAML**: Easy configuration of agents and models
- **Interactive console**: Conversational interface with the agents

## Tutorials

### Creating Your First Agent System

1. **Set up your environment**:

   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```

2. **Create a basic configuration file** named `agent.yaml`:

   ```yaml
   agents:
     root:
       name: assistant
       model: openai_gpt4
       description: A helpful assistant
       instruction: |
         You are a helpful assistant that answers general questions.
       tools:
         - web_browser

   models:
     openai_gpt4:
       type: openai
       model: gpt-4
       temperature: 0.2
       max_tokens: 2048
   ```

3. **Run your agent system**:

   ```bash
   ./cagent -config agent.yaml
   ```

4. **Interact with your agent** in the console.

## How-to Guides

### How to Configure Multiple Agents

Create a configuration with a root agent and specialized sub-agents:

```yaml
agents:
  root:
    name: dev_assistant
    model: openai_gpt4
    description: Software development assistant
    tools:
      - file_system
      - web_browser
    instruction: |
      You are a software development assistant.
      Delegate to specialized agents when needed.
    sub_agents:
      - code_generator
      - debugger

  code_generator:
    name: code_generator
    model: openai_gpt4
    description: Specialized in generating code
    tools:
      - file_system
    instruction: |
      You are a code generation specialist.

  debugger:
    name: debugger
    model: openai_gpt4
    description: Specialized in debugging code
    tools:
      - file_system
    instruction: |
      You are a debugging specialist.

models:
  openai_gpt4:
    type: openai
    model: gpt-4
    temperature: 0.2
    max_tokens: 2048
```

### How to Run a Specific Agent

```bash
./cagent -agent code_generator
```

### How to Start with an Initial Prompt

```bash
./cagent -prompt "Create a Python script to analyze data from CSV files"
```

## Explanation

### Understanding Agent Delegation

In cagent, agents form a hierarchy where specialized knowledge is distributed among sub-agents. When the root agent encounters a request requiring specialized knowledge, it can delegate the conversation to the appropriate sub-agent.

This delegation mechanism allows for:

- Separation of concerns between agents
- Specialized knowledge in specific domains
- Dynamic conversation routing based on user needs

### Agent Communication Flow

1. User interacts with the root agent
2. Root agent processes the request and identifies required expertise
3. If needed, root agent delegates to a specialized sub-agent
4. Sub-agent handles the specialized request
5. Control returns to root agent when the specialized task is complete

## Reference

### Prerequisites

- Go 1.24 or higher
- OpenAI API key

### Configuration Format

#### Top-Level Structure

```yaml
agents: # Define all your agents here
  root:# The main entry point agent
    # agent properties...
  agent1:# Additional specialized agents
    # agent properties...

models: # Define model configurations
  model1:# Model configuration name
    # model properties...
```

#### Agent Properties

| Property    | Description                                    |
| ----------- | ---------------------------------------------- |
| name        | Identifier for the agent                       |
| model       | References a model configuration               |
| description | Short description of the agent's purpose       |
| instruction | Detailed instructions for the agent's behavior |
| tools       | List of tools the agent can use                |
| sub_agents  | List of specialized agents for delegation      |

#### Model Properties

| Property          | Description                 | Range                      |
| ----------------- | --------------------------- | -------------------------- |
| type              | AI provider (e.g., openai)  |                            |
| model             | Specific model to use       | gpt-4, gpt-3.5-turbo, etc. |
| temperature       | Controls randomness         | 0.0-1.0                    |
| max_tokens        | Maximum response length     | Integer                    |
| top_p             | Controls diversity          | 0.0-1.0                    |
| frequency_penalty | Reduces sequence repetition | -2.0-2.0                   |
| presence_penalty  | Reduces topic repetition    | -2.0-2.0                   |

### Command Line Options

| Option  | Description                  |
| ------- | ---------------------------- |
| -config | Specify a configuration file |
| -agent  | Run a specific agent         |
| -prompt | Start with an initial prompt |

For more example configurations, check the examples directory.
