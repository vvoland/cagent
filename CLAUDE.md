# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Development
- `task build` - Build the application binary (depends on `build-web`)
- `task build-web` - Build the frontend React application
- `task test` - Run Go tests (depends on `build-web`)
- `task lint` - Run golangci-lint
- `task link` - Create symlink to ~/bin for easy access

### Docker Operations
- `task build-image` - Build Docker image
- `task build-local` - Build binaries for local platform using Docker
- `task cross` - Build cross-platform binaries using Docker

### Running cagent
- `./bin/cagent run <config.yaml>` - Run agent with configuration
- `./bin/cagent run <config.yaml> -a <agent_name>` - Run specific agent
- `./bin/cagent web -d <config_dir> <session.db>` - Start web interface
- `./bin/cagent tui <config.yaml>` - Start TUI interface
- `./bin/cagent mcp server --port 8080 --path /mcp --agents-dir <config_dir>` - Start MCP server mode
- `./bin/cagent init` - Initialize new project

### MCP Testing
- `cd examples/mcptesting && go run test-mcp-client.go` - Test MCP server functionality
- Test client verifies agent listing, pulling, and invocation via MCP protocol

## Architecture Overview

cagent is a multi-agent AI system with hierarchical agent structure and pluggable tool ecosystem via MCP (Model Context Protocol).

### Core Components

#### ServiceCore Layer (`pkg/servicecore/`)
- **Multi-tenant architecture**: Client-isolated operations ensuring security between different users
- **Transport-agnostic design**: Core business logic independent of MCP/HTTP transport specifics
- **Agent resolution**: File-based and Docker store-based agent discovery with explicit reference formatting
- **Session management**: Per-client session lifecycle with proper resource cleanup
- **Security-first design**: All operations require client ID scoping, preventing cross-client data access

#### Agent System (`pkg/agent/`)
- **Agent struct**: Core abstraction with name, description, instruction, toolsets, models, and sub-agents
- **Hierarchical structure**: Root agents coordinate sub-agents for specialized tasks
- **Tool integration**: Agents have access to built-in tools (think, todo, memory, transfer_task) and external MCP tools
- **Multi-model support**: Agents can use different AI providers (OpenAI, Anthropic, DMR)

#### Runtime System (`pkg/runtime/`)
- **Event-driven architecture**: Streaming responses for real-time interaction
- **Tool execution**: Handles tool calls and coordinates between agents and external tools
- **Session management**: Maintains conversation state and message history
- **Task delegation**: Routes tasks between agents using transfer_task tool

#### Configuration System (`pkg/config/`)
- **YAML-based configuration**: Declarative agent, model, and tool definitions
- **Agent properties**: name, model, description, instruction, sub_agents, toolsets, think, todo, memory, add_date
- **Model providers**: openai, anthropic, dmr with configurable parameters
- **Tool configuration**: MCP tools (local stdio and remote), builtin tools (filesystem, shell)

#### Command Layer (`cmd/root/`)
- **Multiple interfaces**: CLI (`run.go`), Web (`web.go`), TUI (`tui.go`), API (`api.go`), MCP server (`mcp.go`)
- **Interactive commands**: `/exit`, `/reset`, `/eval` during sessions
- **Debug support**: `--debug` flag for detailed logging
- **MCP server mode**: SSE-based transport for external MCP clients like Claude Code

#### MCP Server (`pkg/mcpserver/`)
- **Protocol compliance**: Full MCP specification implementation with SSE transport
- **Tool handlers**: Agent listing, invocation, session management, and Docker image operations
- **Client isolation**: Per-client contexts preventing cross-client interference
- **Structured responses**: Explicit agent_ref formatting for file vs store-based agents

### Tool System (`pkg/tools/`)

#### Built-in Tools
- **think**: Step-by-step reasoning tool
- **todo**: Task list management  
- **memory**: Persistent SQLite-based storage
- **transfer_task**: Agent-to-agent task delegation
- **filesystem**: File operations
- **shell**: Command execution

#### MCP Integration
- **Local MCP servers**: stdio-based tools via command execution
- **Remote MCP servers**: SSE/streamable transport for remote tools
- **Tool filtering**: Optional tool whitelisting per agent

### Key Patterns

#### Agent Configuration
```yaml
agents:
  root:
    name: agent_name
    model: model_ref
    description: purpose
    instruction: detailed_behavior
    sub_agents: [list]
    toolsets: [tool_configs]
    think: boolean
    todo: boolean
    memory: {path: string}
```

#### Task Delegation Flow
1. User → Root Agent
2. Root Agent analyzes request
3. Routes to appropriate sub-agent via transfer_task
4. Sub-agent processes with specialized tools
5. Results flow back through hierarchy

#### Stream Processing
- Models return streaming responses
- Runtime processes chunks and tool calls
- Events emitted for real-time UI updates
- Tool execution integrated into stream flow

## Development Guidelines

### Testing
- Tests located alongside source files (`*_test.go`)
- Run `task test` to execute full test suite
- Web frontend must be built before running Go tests

### Configuration Validation
- All agent references must exist in config
- Model references must be defined
- Tool configurations validated at startup

### Adding New Features
- Follow existing patterns in `pkg/` directories
- Implement proper interfaces for providers and tools
- Add configuration support if needed
- Consider both CLI and web interface impacts

### Web Frontend (`web/`)
- React/TypeScript application with Tailwind CSS and Radix UI components
- Build with `npm run build` or `task build-web` (required before Go build)
- Embedded in Go binary via `embed.go`
- Real-time communication with backend via SSE/WebSocket
- Includes dark mode toggle, syntax highlighting, and responsive design
- Uses Vite for development and bundling

### Key Patterns

#### Agent Reference Formatting
- **File agents**: Use relative path from agents directory (e.g., `agent.yaml`)
- **Store agents**: Use full Docker image reference with tag (e.g., `user/agent:latest`)
- **Explicit agent_ref field**: MCP responses include unambiguous agent reference for tool calls

#### MCP vs HTTP API
- **MCP Server**: Designed for multi-tenant with client isolation, secure by design, recommended for external integrations
- **HTTP API**: Legacy single-tenant mode, no client isolation, used for backward compatibility
- **ServiceCore abstraction**: Shared business logic between both transport layers

#### Current Multi-Tenant Limitation
- **ServiceCore layer**: Fully supports multi-tenant operation with client isolation
- **MCP implementation**: Client ID extraction from MCP context is not yet implemented
- **Current behavior**: All MCP clients use `DEFAULT_CLIENT_ID` ("__global"), making them effectively share sessions
- **Impact**: Multiple MCP clients can see and interact with each other's agent sessions
- **Recommendation**: Use single MCP client per cagent instance until full multi-tenant support is implemented