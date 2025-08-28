# 🤖 `cagent` 🤖

> A powerful, customizable multi-agent system that orchestrates AI agents with
specialized capabilities and tools.



![cagent in action](docs/assets/cagent-run.gif)

## ✨ What is `cagent`? ✨

`cagent` enables you to create and run intelligent agent teams where each agent has
specialized knowledge, tools, and capabilities.  

Think of it as allowing you to quickly build a virtual team of experts that can collaborate to solve complex problems.

### 🎯 Key Features

- **🏗️ Multi-agent architecture** - Create specialized agents for different
  domains
- **🔧 Rich tool ecosystem** - Agents can use external tools and APIs via the MCP
  protocol
- **🔄 Smart delegation** - Agents can automatically route tasks to the most
  suitable specialist
- **📝 YAML configuration** - Declarative model and agent configuration
- **💭 Advanced reasoning** - Built-in "think", "todo" and "memory" tools for
  complex problem solving
- **🌐 Multiple AI providers** - Support for OpenAI, Anthropic, Gemini and DMR ([Docker Model Runner](https://docs.docker.com/ai/model-runner/))

## 🚀 Quick Start 🚀

### Installation & Setup

#### Prebuilt Binaries

Prebuilt binaries for Windows, MacOS and Linux can be found on the releases page of the [project's GitHub repository](https://github.com/docker/cagent/releases)

Once you've downloaded the appropriate bianry for your platform, you may need to give it executable permissions.  

On MacOS and Linux, this can be done with the following command:  

```sh
# linux amd64 build example
chmod +x /path/to/downloads/cagent-linux-amd64
```

You can then rename the binary to `cagent` and configure your `PATH` to be able to find it (configuration varies by platform).

#### Build from source

If you're hacking on `cagent`, or just want to be on the bleeding edge, then building from source is a must.  

Here's what you need to know:

##### Prerequisites

- Go 1.24 or higher
- API key for your chosen AI provider (OpenAI, Anthropic, Gemini, etc.)
- [Task 3.44 or higher](https://taskfile.dev/installation/)
- [`golangci-lint`](https://golangci-lint.run/docs/welcome/install/#binaries`)

##### Build commands

```bash
# Clone and build
git clone https://github.com/docker/cagent.git
cd cagent
task build

# If using the Docker AI Gateway, set this env var or use the `--models-gateway url_to_docker_ai_gateway` CLI flag
export CAGENT_MODELS_GATEWAY=url_to_docker_ai_gateway

# Alternatively, you to need set keys for remote inference services
# Note that these are not needed if you are using Docker AI Gateway

export OPENAI_API_KEY=your_api_key_here    # For OpenAI models
export ANTHROPIC_API_KEY=your_api_key_here # For Anthopic models
export GOOGLE_API_KEY=your_api_key_here    # For Gemini models

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
    model: openai/gpt-5-mini
    description: A helpful AI assistant
    instruction: |
      You are a knowledgeable assistant that helps users with various tasks.
      Be helpful, accurate, and concise in your responses.
```

#### **Set your API keys:**

Based on the models you configure your agents to use, you will need to set the corresponding provider API key accordingly.

```bash
# For OpenAI models
export OPENAI_API_KEY=your_api_key_here

# For Anthropic models
export ANTHROPIC_API_KEY=your_api_key_here

# For Gemini models
export GOOGLE_API_KEY=your_api_key_here
```

#### Run the agent

```bash
cagent run my-agent.yaml
# or specify a different starting agent from the config, useful for agent teams
cagent run my-agent.yaml -a root
```

### Multi agent team example

```yaml
agents:
  root:
    model: claude
    description: "Main coordinator agent that delegates tasks and manages workflow"
    instruction: |
      You are the root coordinator agent. Your job is to:
      1. Understand user requests and break them down into manageable tasks
      2. Delegate appropriate tasks to your helper agent
      3. Coordinate responses and ensure tasks are completed properly
      4. Provide final responses to the user
      When you receive a request, analyze what needs to be done and decide whether to:
      - Handle it yourself if it's simple
      - Delegate to the helper agent if it requires specific assistance
      - Break complex requests into multiple sub-tasks
    sub_agents: ["helper"]

  helper:
    model: claude
    description: "Assistant agent that helps with various tasks as directed by the root agent"
    instruction: |
      You are a helpful assistant agent. Your role is to:
      1. Complete specific tasks assigned by the root agent
      2. Provide detailed and accurate responses
      3. Ask for clarification if tasks are unclear
      4. Report back to the root agent with your results
      
      Focus on being thorough and helpful in whatever task you're given.

models:
  claude:
    provider: anthropic
    model: claude-sonnet-4-0
    max_tokens: 64000
```

## 🎯 Core Concepts

- **Root Agent**: Main entry point that coordinates the system. This represents the first agent you interact with
- **Sub-Agents**: Specialized agents for specific domains or tasks
- **Tools**: External capabilities agents can use via the Model Context Protocol (MCP)
- **Models**: Models agents can be configures to use. They include the AI provider and the model configuration (model to use, max_tokens, temperature, etc.)

## Delegation Flow

1. User interacts with root agent
2. Root agent analyzes the request
3. Root agent can decide to delegate to appropriate sub-agent if specialized knowledge is needed
4. Sub-agent processes the task delegated to it using its tools and expertise, in it's own agentic loop.
5. Results eventually flow back to the root agent and the user

## Quickly generate agents and agent teams with `cagent new`

Using the command `cagent new` you can quickly generate agents or multi agent teams using a single prompt! `cagent` has a built-in agent dedicated to this task.  

To use the feature, you must have an Anthropic API key available in your environment  

`export ANTHROPIC_API_KEY=your_api_key_here`

```sh
$ cagent new

Welcome to cagent! (Ctrl+C to exit)

What should your agent/agent team do? (describe its purpose):

> I need an agent team that does ...
```

## 🔧 Configuration Reference

### Agent Properties

| Property      | Type    | Description                     | Required |
| ------------- | ------- | ------------------------------- | -------- |
| `name`        | string  | Agent identifier                | ✓        |
| `model`       | string  | Model reference                 | ✓        |
| `description` | string  | Agent purpose                   | ✓        |
| `instruction` | string  | Detailed behavior instructions  | ✓        |
| `sub_agents`  | array   | List of sub-agent names         | ✗        |
| `toolsets`    | array   | Available tools                 | ✗        |
| `add_date`    | boolean | Add current date to context     | ✗        |

#### Example

```yaml
agents:
  agent_name:
    model: string       # Model reference
    description: string # Agent purpose
    instruction: string # Detailed behavior instructions
    tools: []           # Available tools (optional)
    sub_agents: []      # Sub-agent names (optional)
    add_date: boolean   # Add current date to context (optional)
```

### Model Properties

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

#### Example

```yaml
models:
  model_name:
    type: string             # Provider: openai, anthropic, google, dmr
    model: string            # Model name: gpt-4o, claude-3-5-sonnet-latest, gemini-2.5-flash, qwen3:4B, ...
    temperature: float       # Randomness (0.0-1.0)
    max_tokens: integer      # Response length limit
    top_p: float             # Nucleus sampling (0.0-1.0)
    frequency_penalty: float # Repetition penalty (0.0-2.0)
    presence_penalty: float  # Topic repetition penalty (0.0-2.0)
    parallel_tool_calls: boolean
```

#### Model Examples

> ⚠️ **NOTE** ⚠️  
> **More model names can be found [here](https://modelname.ai/)**

```yaml

# OpenAI
models:
  openai:
    type: openai
    model: gpt-5-mini

# Anthropic
models:
  claude:
    type: anthropic
    model: claude-sonnet-4-0

# Gemini
models:
  gemini:
    type: google
    model: gemini-2.5-flash

# Docker Model Runner (DMR)
models:
  qwen:
    type: dmr
    model: ai/qwen3
```

### Alloy models

"Alloy models" essentially means using more than one model in the same chat context. Not at the same time, but "randomly" throughout the conversation to try to take advantage of the strong points of each model.

More information on the idea can be found [here](https://xbow.com/blog/alloy-agents)

To have an agent use an alloy model, you can define more than one model in the `model` field, separated by commas.  

Example:

```yaml
agents:
  root:
    model: anthropic/claude-sonnet-4-0,openai/gpt-5-mini
    ...
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
```

**Built-in tools**

```yaml
toolsets:
  - type: filesystem # Grants the agent filesystem access
  - type: think      # Enables the think tool
  - type: todo       # Enable the todo list tool
    shared: boolean  # Should the todo list be shared between agents (optional)
  - type: memory     # Allows the agent to store memories to a local sqlite db
    path: ./mem.db   # Path to the sqlite database for memory storage (optional)
```

## Built-in Tools

Included in `cagent` are a series of built-in tools that can greatly enhance the capabilities of your agents without needing to configure any external MCP tools.  
Lets go into a bit more detail about the built-in tools that agents can use:

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

### Using tools via the Docker MCP Gateway

Using the `docker mcp gateway` command you can configure your agents with a set of MCP tools
delivered straight from Docker's MCP Gateway.

> you can check `docker mcp gateway run --help` for more information on how to use that command

In this example, lets configure duckduckgo to give our agents the ability to search the web:

```yaml
toolsets:
  - type: mcp
    command: docker
    args: ["mcp", "gateway", "run", "--servers=duckduckgo"]
```

## Pushing and pulling agents and teams from Docker Hub

### `cagent push`

Agent configurations can be packaged and shared to Docker Hub using the `cagent push` command

```sh
cagent push ./<agent-file>.yaml namespace/reponame
```

`cagent` will automatically build an OCI image and push it to the desired repository using your Docker credentials

### `cagent pull`

Pulling agents/teams from Docker Hub is also just one `cagent pull` command away.

```sh
cagent pull agentcatalog/pirate
```

`cagent` will pull the image, extract the yaml file and place it in your working directory for ease of use.

`cagent run agentcatalog_pirate.yaml` will run your newly pulled agent

## CLI Interactive Commands

During CLI sessions, you can use special commands:

| Command  | Description                              |
| -------- | ---------------------------------------- |
| `/exit`  | Exit the program                         |
| `/reset` | Clear conversation history               |
| `/eval`  | Save current conversation for evaluation |


## 🤝 Examples

Explore the [examples/config](examples/config/) directory for ready-to-use configurations:

- [examples/config/agent.yaml](examples/config/agent.yaml) - Basic assistant
- [examples/config/code.yaml](examples/config/code.yaml) - Software development team
- [examples/config/finance.yaml](examples/config/finance.yaml) - Financial analysis specialist
