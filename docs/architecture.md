# Architecture Guide

This guide explains the internal architecture of cagent, how components
interact, and the design principles behind the system.

## System Overview

cagent is built as a modular, event-driven multi-agent system with the following
key characteristics:

- **Hierarchical Agent Structure**: Agents can have sub-agents for specialized
  tasks
- **Event-Driven Runtime**: Streaming architecture for real-time interactions
- **Pluggable Tools**: Extensible tool system via Model Context Protocol (MCP)
- **Provider Agnostic**: Support for multiple AI providers
- **Configuration-Driven**: YAML-based declarative configuration

## Architecture Diagram

```mermaid
graph TB
    subgraph "User Interface Layer"
        CLI[CLI Interface]
        WEB[Web Interface]
        UI[Desktop UI]
        API[REST API]
    end

    subgraph "Command Layer"
        ROOT[Root Command]
        RUN[Run Command]
        INIT[Init Command]
        EVAL[Eval Command]
    end

    subgraph "Core Runtime"
        RT[Runtime]
        SESS[Session Manager]
        TEAM[Team Coordinator]
    end

    subgraph "Agent Layer"
        AGENT[Agent]
        TOOLS[Tool System]
        MEMORY[Memory Manager]
    end

    subgraph "Model Layer"
        OPENAI[OpenAI Provider]
        ANTHROPIC[Anthropic Provider]
        DMR[DMR Provider]
    end

    subgraph "Configuration"
        CONFIG[Config Loader]
        VALIDATOR[Validator]
        LOADER[Agent Loader]
    end

    subgraph "External Tools"
        MCP[MCP Tools]
        SHELL[Shell Tools]
        BUILTIN[Built-in Tools]
    end

    CLI --> ROOT
    WEB --> ROOT
    UI --> ROOT
    API --> ROOT

    ROOT --> RUN
    ROOT --> INIT
    ROOT --> EVAL

    RUN --> RT
    RT --> SESS
    RT --> TEAM

    TEAM --> AGENT
    AGENT --> TOOLS
    AGENT --> MEMORY

    AGENT --> OPENAI
    AGENT --> ANTHROPIC
    AGENT --> DMR

    RT --> CONFIG
    CONFIG --> VALIDATOR
    CONFIG --> LOADER

    TOOLS --> MCP
    TOOLS --> SHELL
    TOOLS --> BUILTIN
```

## Component Architecture

### 1. Command Layer (`cmd/root/`)

The command layer provides multiple interfaces for interacting with cagent:

#### Root Command (`root.go`)

- Entry point for all CLI operations
- Manages global flags and configuration
- Dispatches to appropriate subcommands

#### Run Command (`run.go`)

- Interactive chat interface
- Handles user input and agent responses
- Manages conversation flow and session state

#### Web Interface (`web.go`)

- HTTP server for web-based interactions
- RESTful API endpoints
- WebSocket support for real-time communication

#### TUI Command (`tui.go`)

- Desktop application interface
- Native GUI components
- Cross-platform compatibility

### 2. Configuration System (`pkg/config/`)

The configuration system handles agent and model definitions:

```mermaid
graph LR
    YAML[YAML File] --> LOADER[Config Loader]
    LOADER --> VALIDATOR[Validator]
    VALIDATOR --> CONFIG[Config Object]
    CONFIG --> AGENT_FACTORY[Agent Factory]
    AGENT_FACTORY --> AGENTS[Agent Instances]
```

#### Configuration Loading Flow

1. **Parse YAML**: Load and parse configuration file
2. **Validate Structure**: Check syntax and required fields
3. **Cross-Reference**: Ensure all references are valid
4. **Create Objects**: Instantiate agents and models

### 3. Runtime System (`pkg/runtime/`)

The runtime system is the core execution engine:

```mermaid
sequenceDiagram
    participant User
    participant Runtime
    participant Agent
    participant Model
    participant Tools

    User->>Runtime: Send Message
    Runtime->>Agent: Get Tools
    Agent->>Runtime: Return Tools
    Runtime->>Model: Create Stream
    Model->>Runtime: Stream Response
    Runtime->>User: Stream Events

    alt Tool Call Required
        Runtime->>Tools: Execute Tool
        Tools->>Runtime: Return Result
        Runtime->>Model: Continue Stream
        Model->>Runtime: Stream Response
    end

    Runtime->>User: Final Response
```

#### Key Components

**Runtime Engine** (`runtime.go`):

- Manages agent lifecycle
- Handles streaming responses
- Coordinates tool execution
- Manages task delegation

**Event System**:

- Real-time streaming architecture
- Multiple event types for different actions
- Asynchronous processing

**Tool Integration**:

- Dynamic tool discovery
- Tool call parsing and execution
- Error handling and recovery

### 4. Agent System (`pkg/agent/`)

Agents are the core abstraction in cagent:

```mermaid
classDiagram
    class Agent {
        +name: string
        +description: string
        +instruction: string
        +model: Provider
        +tools: []ToolSet
        +subAgents: []*Agent
        +parents: []*Agent
        +memoryManager: Manager

        +Tools() []Tool
        +HasSubAgents() bool
    }

    class ToolSet {
        +Tools(ctx) []Tool
    }

    class Provider {
        +CreateChatCompletionStream(ctx, messages, tools) Stream
    }

    Agent --> ToolSet
    Agent --> Provider
    Agent --> Agent : sub-agents
```

#### Agent Lifecycle

1. **Creation**: Agent instantiated from configuration
2. **Initialization**: Tools and sub-agents connected
3. **Execution**: Processing messages and tool calls
4. **Delegation**: Task transfer to sub-agents
5. **Cleanup**: Resource cleanup and state persistence

### 5. Model Integration (`pkg/model/`)

The model layer abstracts different AI providers:

```mermaid
graph TB
    subgraph "Provider Interface"
        INTERFACE[Provider Interface]
    end

    subgraph "Implementations"
        OPENAI_IMPL[OpenAI Implementation]
        ANTHROPIC_IMPL[Anthropic Implementation]
        DMR_IMPL[DMR Implementation]
    end

    subgraph "Common Features"
        STREAMING[Streaming Support]
        TOOLS_SUPPORT[Tool Calling]
        CONFIG[Configuration]
    end

    INTERFACE --> OPENAI_IMPL
    INTERFACE --> ANTHROPIC_IMPL
    INTERFACE --> DMR_IMPL

    OPENAI_IMPL --> STREAMING
    ANTHROPIC_IMPL --> STREAMING
    DMR_IMPL --> STREAMING

    OPENAI_IMPL --> TOOLS_SUPPORT
    ANTHROPIC_IMPL --> TOOLS_SUPPORT
    DMR_IMPL --> TOOLS_SUPPORT
```

#### Provider Interface

All providers implement a common interface:

- `CreateChatCompletionStream()`: Stream-based chat completion
- Model-specific configuration handling
- Tool call support
- Error handling and retry logic

### 6. Tool System (`pkg/tools/`)

The tool system provides extensible capabilities:

```mermaid
graph TB
    subgraph "Tool Types"
        BUILTIN[Built-in Tools]
        MCP_TOOLS[MCP Tools]
        SHELL_TOOLS[Shell Tools]
    end

    subgraph "Built-in Tools"
        THINK[Think Tool]
        TODO[Todo Tool]
        MEMORY[Memory Tool]
        TRANSFER[Transfer Task Tool]
    end

    subgraph "MCP Protocol"
        MCP_CLIENT[MCP Client]
        EXTERNAL_TOOLS[External Tools]
    end

    BUILTIN --> THINK
    BUILTIN --> TODO
    BUILTIN --> MEMORY
    BUILTIN --> TRANSFER

    MCP_TOOLS --> MCP_CLIENT
    MCP_CLIENT --> EXTERNAL_TOOLS
```

#### Tool Execution Flow

1. **Discovery**: Agent discovers available tools
2. **Registration**: Tools registered with runtime
3. **Invocation**: Model decides to call tool
4. **Execution**: Tool handler processes request
5. **Response**: Result returned to model

### 7. Session Management (`pkg/session/`)

Session management handles conversation state:

```mermaid
graph LR
    subgraph "Session Components"
        SESS[Session]
        MESSAGES[Message History]
        METADATA[Metadata]
    end

    subgraph "Message Types"
        USER[User Messages]
        AGENT[Agent Messages]
        TOOL[Tool Messages]
        SYSTEM[System Messages]
    end

    SESS --> MESSAGES
    SESS --> METADATA
    MESSAGES --> USER
    MESSAGES --> AGENT
    MESSAGES --> TOOL
    MESSAGES --> SYSTEM
```

#### Session Features

- **Message History**: Complete conversation tracking
- **Agent Context**: Per-agent message filtering
- **Persistence**: Session state can be saved/loaded
- **Metadata**: Additional context and configuration

### 8. Team Coordination (`pkg/team/`)

The team system manages multi-agent coordination:

```mermaid
graph TB
    subgraph "Team Structure"
        TEAM[Team Manager]
        AGENTS[Agent Registry]
        DELEGATION[Task Delegation]
    end

    subgraph "Agent Hierarchy"
        ROOT[Root Agent]
        SUB1[Sub-Agent 1]
        SUB2[Sub-Agent 2]
        SUB3[Sub-Agent 3]
    end

    TEAM --> AGENTS
    TEAM --> DELEGATION

    AGENTS --> ROOT
    ROOT --> SUB1
    ROOT --> SUB2
    ROOT --> SUB3
```

#### Delegation Flow

1. **Task Analysis**: Root agent analyzes incoming task
2. **Agent Selection**: Identifies best sub-agent for task
3. **Context Transfer**: Passes relevant context
4. **Execution**: Sub-agent processes task
5. **Result Integration**: Results merged back to main conversation

## Data Flow

### Message Processing Flow

```mermaid
sequenceDiagram
    participant User
    participant Runtime
    participant Session
    participant Agent
    participant Model
    participant Tools

    User->>Runtime: Input Message
    Runtime->>Session: Add User Message
    Session->>Agent: Get Context Messages
    Agent->>Session: Filtered Messages
    Runtime->>Model: Create Stream

    loop Stream Processing
        Model->>Runtime: Stream Chunk
        Runtime->>User: Stream Event

        alt Tool Call
            Runtime->>Tools: Execute Tool
            Tools->>Runtime: Tool Result
            Runtime->>Session: Add Tool Message
        end
    end

    Runtime->>Session: Add Agent Response
    Session->>Session: Update History
```

### Configuration Loading Flow

```mermaid
sequenceDiagram
    participant CLI
    participant Loader
    participant Config
    participant Validator
    participant Factory
    participant Team

    CLI->>Loader: Load Config File
    Loader->>Config: Parse YAML
    Config->>Validator: Validate Structure
    Validator->>Config: Validation Result
    Config->>Factory: Create Agents
    Factory->>Team: Register Agents
    Team->>CLI: Ready Team
```

## Design Principles

### 1. Modularity

- Each component has a single responsibility
- Clear interfaces between components
- Pluggable architecture for extensibility

### 2. Event-Driven Architecture

- Streaming responses for real-time interaction
- Asynchronous processing where possible
- Event-based communication between components

### 3. Configuration-Driven

- Declarative agent and model definitions
- No hard-coded behaviors
- Easy to modify and extend

### 4. Provider Agnostic

- Abstract interface for AI providers
- Consistent behavior across providers
- Easy to add new providers

### 5. Tool Extensibility

- Standard tool interface
- Support for external tools via MCP
- Built-in tools for common operations

## Performance Considerations

### 1. Streaming Architecture

- Reduces latency for user interactions
- Enables real-time progress feedback
- Efficient memory usage

### 2. Lazy Loading

- Agents created only when needed
- Tools loaded on demand
- Configuration validated incrementally

### 3. Resource Management

- Proper cleanup of resources
- Connection pooling for providers
- Memory management for large conversations

### 4. Concurrent Processing

- Multiple agents can run concurrently
- Tool calls executed asynchronously
- Efficient context switching

## Security Considerations

### 1. API Key Management

- Environment variable usage
- No hardcoded credentials
- Secure key rotation support

### 2. Tool Permissions

- Granular tool access control
- Filesystem permission restrictions
- Shell command filtering

### 3. Input Validation

- Configuration validation
- User input sanitization
- Tool parameter validation

### 4. Audit Logging

- Complete action logging
- Debug information capture
- Error tracking and reporting

## Extension Points

### 1. Custom Providers

Implement the `Provider` interface to add new AI providers:

```go
type Provider interface {
    CreateChatCompletionStream(ctx context.Context, messages []Message, tools []Tool) (Stream, error)
}
```

### 2. Custom Tools

Create new tools by implementing the `Tool` interface:

```go
type Tool interface {
    Function() Function
    Handler(ctx context.Context, call ToolCall) (*ToolCallResult, error)
}
```

### 3. Custom Memory Managers

Implement custom memory strategies:

```go
type Manager interface {
    Store(ctx context.Context, key string, value interface{}) error
    Retrieve(ctx context.Context, key string) (interface{}, error)
}
```

### 4. Custom UI Components

Add new interfaces by implementing command handlers:

```go
func NewCustomCmd() *cobra.Command {
    return &cobra.Command{
        Use: "custom",
        RunE: customCommandHandler,
    }
}
```

This architecture provides a solid foundation for building sophisticated
multi-agent systems while maintaining flexibility and extensibility.
