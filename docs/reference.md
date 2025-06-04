# Configuration Reference

Complete reference documentation for all cagent configuration options, parameters, and settings. Use this as your definitive guide for YAML configuration syntax and available options.

## Configuration File Structure

cagent uses YAML configuration files with two main sections:

```yaml
agents:
  # Agent definitions - defines behavior and capabilities
  root:
    # Root agent configuration (required)
  agent_name:
    # Sub-agent configurations (optional)

models:
  # Model configurations - defines AI providers and parameters
  model_name:
    # Model provider settings
```

## Configuration Validation

cagent validates configurations on startup. Common validation errors:

- Missing required fields (`name`, `model`, `description`, `instruction`)
- Invalid model references (agent references non-existent model)
- Circular sub-agent dependencies
- Invalid temperature/token values

## 📊 Agents Section

The `agents` section defines all agents in your system. The `root` agent is required and serves as the main entry point.

### Agent Configuration Fields

#### Required Fields

| Field         | Type   | Description                                        | Example                    |
| ------------- | ------ | -------------------------------------------------- | -------------------------- |
| `name`        | String | Unique identifier for the agent (3-50 characters)  | `"research_assistant"`     |
| `model`       | String | Reference to a model defined in models section     | `"gpt4"`                   |
| `description` | String | Brief description of agent purpose (max 200 chars) | `"AI research specialist"` |
| `instruction` | String | Detailed behavioral instructions (multi-line YAML) | See instruction examples   |

#### Optional Fields

| Field        | Type    | Default | Description                                |
| ------------ | ------- | ------- | ------------------------------------------ |
| `toolsets`   | Array   | `[]`    | List of toolset configurations             |
| `sub_agents` | Array   | `[]`    | Names of agents this agent can delegate to |
| `think`      | Boolean | `false` | Enable metacognitive thinking capabilities |
| `add_date`   | Boolean | `false` | Include current date in agent context      |

### Agent Configuration Examples

#### Basic Agent

```yaml
agents:
  root:
    name: assistant
    model: gpt4
    description: General purpose AI assistant
    instruction: |
      You are a helpful, knowledgeable assistant. Provide accurate,
      well-structured responses to user questions.
```

#### Agent with Tools

```yaml
agents:
  root:
    name: researcher
    model: claude
    description: Research agent with web search capabilities
    instruction: |
      You are a research specialist. Use web search to find current
      information and always cite your sources.
    toolsets:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
        tools: ["search", "summarize"] # Optional: filter specific tools
    think: true
```

#### Agent with Sub-Agents

```yaml
agents:
  root:
    name: coordinator
    model: gpt4
    description: Multi-agent coordinator
    instruction: |
      You coordinate a team of specialists. Delegate tasks based on
      the type of request:
      - Technical questions → tech_specialist
      - Creative work → creative_specialist
    sub_agents: [tech_specialist, creative_specialist]

  tech_specialist:
    name: tech_specialist
    model: claude
    description: Technical expert for engineering questions
    instruction: |
      You are a senior software engineer specializing in system
      architecture, coding best practices, and technical solutions.

  creative_specialist:
    name: creative_specialist
    model: gpt4
    description: Creative specialist for content and design
    instruction: |
      You are a creative professional specializing in content creation,
      copywriting, and creative problem solving.
```

### Instruction Field Best Practices

The `instruction` field is the most critical part of agent configuration. Structure it clearly:

```yaml
instruction: |
  You are a [ROLE] with expertise in [DOMAIN].

  **Your Responsibilities:**
  - Primary responsibility 1
  - Primary responsibility 2

  **Workflow:**
  1. Step-by-step process
  2. Decision points and criteria

  **Tools Available:** (if applicable)
  - toolset_name: Toolset description and available tools

  **When to Delegate:** (if sub-agents available)
  - Task type → sub_agent_name

  **Constraints:**
  - Behavioral limitations
  - Output format requirements
  - Quality standards
```

### Sub-Agent Configuration

Sub-agents have identical configuration options to the root agent:

```yaml
agents:
  root:
    name: manager
    # ... configuration
    sub_agents: [specialist1, specialist2]

  specialist1:
    name: specialist1
    model: claude
    description: Domain specialist 1
    instruction: |
      You are a specialist in [specific domain].
      Focus only on tasks within your expertise.
    # Can have their own tools and sub-agents
    tools: []
    sub_agents: []
```

## 🤖 Models Section

The `models` section defines AI providers and their configurations. Each model configuration specifies the provider, model variant, and generation parameters.

### Model Configuration Fields

#### Required Fields

| Field   | Type   | Description            | Valid Values               |
| ------- | ------ | ---------------------- | -------------------------- |
| `type`  | String | AI provider            | `openai`, `anthropic`      |
| `model` | String | Specific model variant | See supported models below |

#### Optional Fields

| Field               | Type    | Default | Range     | Description                                 |
| ------------------- | ------- | ------- | --------- | ------------------------------------------- |
| `temperature`       | Float   | `0.7`   | `0.0-1.0` | Controls randomness/creativity in responses |
| `max_tokens`        | Integer | `2048`  | `1-32768` | Maximum response length in tokens           |
| `top_p`             | Float   | `1.0`   | `0.0-1.0` | Nucleus sampling threshold                  |
| `frequency_penalty` | Float   | `0.0`   | `0.0-2.0` | Reduces repetition of frequent tokens       |
| `presence_penalty`  | Float   | `0.0`   | `0.0-2.0` | Reduces repetition of any tokens            |

### Supported Models

#### OpenAI Models

| Model Name      | Description                           | Context Window | Cost |
| --------------- | ------------------------------------- | -------------- | ---- |
| `gpt-4o`        | Latest GPT-4 with vision capabilities | 128k tokens    | $$   |
| `gpt-4-turbo`   | Fast, capable GPT-4 variant           | 128k tokens    | $$   |
| `gpt-3.5-turbo` | Fast, cost-effective option           | 16k tokens     | $    |

#### Anthropic Models

| Model Name                 | Description                 | Context Window | Cost |
| -------------------------- | --------------------------- | -------------- | ---- |
| `claude-3-5-sonnet-latest` | Best reasoning and analysis | 200k tokens    | $$   |
| `claude-3-haiku`           | Fast, efficient option      | 200k tokens    | $    |

### Model Configuration Examples

#### High Creativity Configuration

```yaml
models:
  creative_writer:
    type: openai
    model: gpt-4o
    temperature: 0.9 # High creativity
    max_tokens: 4000 # Long responses
    top_p: 0.9 # Diverse vocabulary
    frequency_penalty: 0.3 # Avoid repetition
```

#### Analytical Configuration

```yaml
models:
  data_analyst:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.2 # Low creativity, high consistency
    max_tokens: 8000 # Detailed analysis
    top_p: 0.95 # Focused vocabulary
```

#### Balanced Configuration

```yaml
models:
  general_assistant:
    type: openai
    model: gpt-4o
    temperature: 0.7 # Balanced creativity
    max_tokens: 2000 # Standard length
    # Other parameters use defaults
```

#### Budget-Conscious Configuration

```yaml
models:
  budget_model:
    type: openai
    model: gpt-3.5-turbo # Most cost-effective
    temperature: 0.5
    max_tokens: 1000 # Shorter responses = lower cost
```

### Parameter Tuning Guidelines

#### Temperature Selection

- **0.0-0.3**: Factual, consistent, deterministic responses
- **0.4-0.7**: Balanced creativity and consistency
- **0.8-1.0**: High creativity, more varied responses

#### Token Limits by Use Case

- **500-1000**: Brief responses, chat interactions
- **1000-2000**: Standard responses, explanations
- **2000-4000**: Detailed analysis, documentation
- **4000+**: Comprehensive reports, long-form content

#### Top-P Usage

- **0.1-0.5**: Very focused, predictable vocabulary
- **0.6-0.9**: Balanced vocabulary diversity
- **0.95-1.0**: Maximum vocabulary diversity

### Multiple Model Strategy

Use different models for different purposes:

```yaml
models:
  fast_chat:
    type: openai
    model: gpt-3.5-turbo
    temperature: 0.7
    max_tokens: 1000

  deep_analysis:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.3
    max_tokens: 8000

  creative_content:
    type: openai
    model: gpt-4o
    temperature: 0.8
    max_tokens: 4000

agents:
  root:
    model: fast_chat # Use fast model for coordination
    sub_agents: [analyst, writer]

  analyst:
    model: deep_analysis # Use analytical model for analysis

  writer:
    model: creative_content # Use creative model for writing
```

## 🔧 Tools Section

Tools extend agent capabilities beyond language processing by connecting to external systems, APIs, and services through the Model Context Protocol (MCP).

### Tool Configuration Fields

| Field     | Type   | Required | Description                                |
| --------- | ------ | -------- | ------------------------------------------ |
| `type`    | String | Yes      | Tool type (currently only `mcp` supported) |
| `command` | String | Yes      | Command or executable to run               |
| `args`    | Array  | No       | Arguments passed to the command            |

### Tool Configuration Examples

#### Web Search Tool

```yaml
tools:
  - type: mcp
    command: npx
    args: ["-y", "@modelcontextprotocol/server-brave-search"]
```

#### File System Tool

```yaml
tools:
  - type: mcp
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
```

#### Database Tool

```yaml
tools:
  - type: mcp
    command: npx
    args: ["-y", "@modelcontextprotocol/server-sqlite"]
```

#### Custom Docker Tool

```yaml
tools:
  - type: mcp
    command: docker
    args:
      - "run"
      - "-i"
      - "--rm"
      - "my-custom-tool:latest"
      - "tool-specific-args"
```

### Available MCP Tools

#### Official MCP Tools

| Tool        | NPM Package                                 | Description               |
| ----------- | ------------------------------------------- | ------------------------- |
| Web Search  | `@modelcontextprotocol/server-brave-search` | Brave search integration  |
| File System | `@modelcontextprotocol/server-filesystem`   | Local file operations     |
| SQLite      | `@modelcontextprotocol/server-sqlite`       | SQLite database access    |
| Git         | `@modelcontextprotocol/server-git`          | Git repository operations |

#### Third-Party MCP Tools

| Tool          | NPM Package                             | Description                |
| ------------- | --------------------------------------- | -------------------------- |
| Airbnb Search | `@openbnb/mcp-server-airbnb`            | Airbnb listing search      |
| GitHub        | `@modelcontextprotocol/server-github`   | GitHub API integration     |
| Postgres      | `@modelcontextprotocol/server-postgres` | PostgreSQL database access |

### Tool Usage in Agent Instructions

Always document available tools in agent instructions:

```yaml
agents:
  root:
    instruction: |
      You have access to the following tools:

      **Web Search**
      - search(query: str) -> SearchResults
      - Use for current information and research

      **File Operations**  
      - read_file(path: str) -> str
      - write_file(path: str, content: str) -> None
      - list_files(path: str) -> List[str]

      **When to Use Tools:**
      - Search for information not in your training data
      - Access or modify files when requested
      - Always explain your tool usage to the user
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-filesystem"]
```

### Special Features

#### Think Tool

The think tool is enabled per-agent, not configured in the tools array:

```yaml
agents:
  root:
    name: analytical_agent
    think: true # Enables metacognitive reasoning
    instruction: |
      Use the think tool for complex problems:
      1. Break down the problem
      2. Consider multiple approaches  
      3. Validate your reasoning
      4. Present your conclusion
```

#### Date Context

Add current date to agent context:

```yaml
agents:
  root:
    add_date: true # Includes current date in context
    instruction: |
      You have access to the current date and can reference it
      when discussing time-sensitive topics.
```

### Tool Security Considerations

- **Filesystem Access**: Limit file operations to specific directories
- **Network Access**: Be cautious with tools that make external requests
- **Command Execution**: Validate and sanitize any user inputs passed to tools
- **Docker Tools**: Use official images and limit resource access

```yaml
# Example: Restricted file access
tools:
  - type: mcp
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-filesystem"
      - "--allowed-directory"
      - "/safe/directory/path"
```

### Troubleshooting Tools

#### Tool Not Working

1. **Check tool installation**: Ensure MCP server is available
2. **Verify command path**: Confirm command exists and is executable
3. **Check arguments**: Validate argument syntax and values
4. **Review logs**: Check cagent logs for tool execution errors

#### Agent Not Using Tools

1. **Update instructions**: Explicitly mention tool availability and usage
2. **Provide examples**: Show how and when to use specific tools
3. **Set expectations**: Clearly state tool capabilities and limitations

```yaml
# Example: Explicit tool usage instructions
instruction: |
  IMPORTANT: Always use web search for questions about:
  - Current events or recent developments
  - Statistical data or market information
  - Technical specifications or product details

  Before answering, ask yourself: "Do I need current information?"
  If yes, use the search tool first.
```

## 📝 Complete Configuration Example

Here's a comprehensive example showcasing all configuration options:

```yaml
# Complete cagent configuration example
agents:
  root:
    name: ai_research_coordinator
    model: coordinator_model
    description: AI research coordination system with specialized teams
    instruction: |
      You are an AI research coordinator managing a team of specialists.

      **Your Role:**
      - Coordinate research projects across multiple domains
      - Delegate to appropriate specialists based on task requirements
      - Synthesize findings from multiple sources
      - Ensure research quality and accuracy

      **Delegation Strategy:**
      - Literature reviews → research_specialist
      - Data analysis → data_analyst  
      - Writing tasks → technical_writer
      - Fact verification → fact_checker

      **Available Tools:**
      - Web search for current research and papers
      - File operations for document management

      **Quality Standards:**
      - Always verify information through multiple sources
      - Cite all sources using academic format
      - Provide balanced, objective analysis
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-filesystem"]
    sub_agents:
      [research_specialist, data_analyst, technical_writer, fact_checker]
    think: true
    add_date: true

  research_specialist:
    name: research_specialist
    model: research_model
    description: Specialist in academic research and literature analysis
    instruction: |
      You are an academic research specialist with expertise in:
      - Literature review and synthesis
      - Research methodology evaluation
      - Academic source identification and validation

      **Research Process:**
      1. Define research scope and objectives
      2. Conduct comprehensive literature search
      3. Evaluate source credibility and relevance
      4. Synthesize findings into coherent analysis
      5. Identify gaps and future research directions
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]

  data_analyst:
    name: data_analyst
    model: analytical_model
    description: Data analysis and statistical interpretation specialist
    instruction: |
      You are a data analysis specialist focusing on:
      - Statistical analysis and interpretation
      - Data visualization and presentation
      - Methodology validation
      - Quantitative research support

      Always explain statistical concepts clearly and highlight
      limitations and assumptions in your analysis.
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-sqlite"]

  technical_writer:
    name: technical_writer
    model: writing_model
    description: Technical writing and documentation specialist
    instruction: |
      You are a technical writing specialist who creates:
      - Research reports and documentation
      - Technical summaries and abstracts
      - Academic papers and presentations

      **Writing Standards:**
      - Clear, concise, and accessible language
      - Proper academic formatting and citations
      - Logical structure and flow
      - Audience-appropriate technical depth
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-filesystem"]

  fact_checker:
    name: fact_checker
    model: verification_model
    description: Fact verification and accuracy validation specialist
    instruction: |
      You are a fact-checking specialist responsible for:
      - Verifying claims and assertions
      - Cross-referencing multiple sources
      - Identifying potential biases or conflicts of interest
      - Assessing source credibility and reliability

      **Verification Process:**
      1. Identify specific claims requiring verification
      2. Search for authoritative, primary sources
      3. Cross-reference across multiple independent sources
      4. Flag any unverifiable or conflicting information
      5. Provide confidence ratings for verified facts
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
    think: true

models:
  coordinator_model:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.5
    max_tokens: 3000
    top_p: 0.95

  research_model:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.3
    max_tokens: 4000
    frequency_penalty: 0.1

  analytical_model:
    type: openai
    model: gpt-4o
    temperature: 0.2
    max_tokens: 3000
    top_p: 0.9

  writing_model:
    type: openai
    model: gpt-4o
    temperature: 0.6
    max_tokens: 4000
    frequency_penalty: 0.2
    presence_penalty: 0.1

  verification_model:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.1
    max_tokens: 2000
```

## 🚀 Quick Reference Card

### Minimal Configuration

```yaml
agents:
  root:
    name: assistant
    model: gpt4
    description: General assistant
    instruction: "You are a helpful assistant."

models:
  gpt4:
    type: openai
    model: gpt-4o
```

### Common Field Values

```yaml
# Agent fields
name: "string (3-50 chars)"
model: "model_reference"
description: "string (max 200 chars)"
think: true|false
add_date: true|false

# Model fields
type: "openai"|"anthropic"
model: "gpt-4o"|"claude-3-5-sonnet-latest"
temperature: 0.0-1.0
max_tokens: 1-32768
top_p: 0.0-1.0
frequency_penalty: 0.0-2.0
presence_penalty: 0.0-2.0

# Tool fields
type: "mcp"
command: "executable_name"
args: ["arg1", "arg2"]
```

## 📚 Additional Resources

- **[Getting Started Tutorial](./tutorial.md)** - Build your first agent
- **[How-to Guide](./howto.md)** - Practical configuration examples
- **[Explanation](./explanation.md)** - Architecture and concepts
- **[Examples Directory](../examples/)** - Ready-to-use configurations
