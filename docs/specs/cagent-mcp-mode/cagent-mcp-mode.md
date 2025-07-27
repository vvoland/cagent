# cagent MCP Server Mode Design

## Overview

This document outlines the design for an MCP (Model Context Protocol) server mode for cagent, allowing external clients like Claude Code to programmatically invoke cagent agents and maintain conversational sessions.

## Architectural Decision Process

### Discovery of Existing HTTP API

During analysis, we discovered that cagent already has a functioning HTTP API (`pkg/server/`) that provides:
- Agent management and execution
- Session creation and management
- Docker image support via content store
- Streaming responses

### Critical Security Analysis

**However, the existing HTTP API has fundamental security flaws:**
- **Zero client isolation** - any client can see all sessions from all other clients
- **No authentication or authorization** - anonymous access to all operations
- **Cross-client session manipulation** - clients can delete or hijack other clients' sessions
- **Shared runtime instances** - no separation between different clients

**Example Attack Scenarios:**
```bash
# Any client can list ALL sessions from ALL other clients
curl http://localhost:8080/api/sessions

# Any client can delete any other client's session
curl -X DELETE http://localhost:8080/api/sessions/other-client-session-id

# Any client can send messages to any other client's session
curl -X POST http://localhost:8080/api/sessions/hijacked-session/agent/reviewer
```

### Architecture Options Evaluated

#### Initial Options
- **Option 1**: Build MCP Server from Scratch
- **Option 2**: Add MCP Transport to Existing HTTP Server

#### Extended Options (After HTTP API Discovery)
- **Option 2a**: Direct MCP integration with existing HTTP server
- **Option 2b**: Retrofit security into existing HTTP server
- **Option 2c**: Extract shared logic into common core layer

### **Selected Approach: Option 1 + Shared Core (Hybrid)**

**Why This Approach:**

1. **Security Requirements**: MCP needs proper client isolation from day 1
   - Existing HTTP API is fundamentally insecure for multi-tenant use
   - Retrofitting security would be more complex than building correctly

2. **Architecture Benefits**: Shared core provides foundation for both transports
   - Extract working agent execution logic into `pkg/servicecore/`
   - Build secure MCP server using servicecore
   - Future: Refactor HTTP API to use same secure servicecore

3. **Incremental Value**: Enables future HTTP API security improvements
   - Phase 1: Build secure MCP with servicecore
   - Phase 2: Migrate HTTP API to use same secure core
   - Result: Both transports benefit from battle-tested, multi-tenant core

4. **Code Reuse**: Leverages existing working components
   - Agent loading (`pkg/loader`)
   - Runtime execution (`pkg/runtime`)
   - Content store operations (`pkg/content`)
   - Docker image resolution patterns

**Final Architecture:**
```
┌─────────────────┐    ┌─────────────────┐
│   MCP Server    │    │  HTTP Server    │
│  (Transport)    │    │  (Transport)    │  <- Future refactor
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          └──────────┬───────────┘
                     │
        ┌────────────▼─────────────┐
        │     Service Core         │  <- Shared business logic
        │   (Multi-tenant)         │     with client isolation
        └────────────┬─────────────┘
                     │
        ┌────────────▼─────────────┐
        │   Existing Components    │
        │  (runtime, loader, etc.) │
        └──────────────────────────┘
```

## Architecture Options

### Option 1: One-Shot Mode vs Session-Based Mode

#### One-Shot Mode (Stateless)
Each MCP call creates a fresh cagent runtime instance:
- Load agent configuration
- Process single message  
- Return response
- Terminate runtime

**Pros:** Simple implementation, stateless, no memory accumulation
**Cons:** High overhead, no conversation context, inefficient for multi-turn

#### Session-Based Mode (Stateful) - **SELECTED**
MCP server maintains persistent cagent sessions:
- Sessions map to long-running cagent runtime instances
- Conversation state preserved across calls
- Multi-turn interactions supported
- Session lifecycle management

**Pros:** Efficient resource usage, true conversational flow, supports agent memory
**Cons:** Complex session management, memory growth, cleanup complexity

**Decision:** Selected session-based mode for rich conversational interactions.

## Proposed Implementation

### Command Structure
```bash
cagent mcp run [--port 8080] [--host localhost] [--timeout 3600]
```

### MCP Tools Exposed

#### Core Tools
```typescript
// Start new session with specified agent
create_agent_session(agent: string, session_name?: string) -> {agent_session_id: string}

// Send message to existing session  
send_message(agent_session_id: string, message: string) -> {response: string, events: Event[]}

// One-shot invocation (stateless)
invoke_agent(agent: string, message: string) -> {response: string}

// List available agents from files and store
list_agents(source?: string) -> {agents: AgentInfo[]}

// Session management
get_agent_session_info(agent_session_id: string) -> {agent: string, message_count: number, created: string}
list_agent_sessions() -> {sessions: SessionInfo[]}
close_agent_session(agent_session_id: string) -> {success: boolean}

// Docker image operations
pull_agent(registry_ref: string) -> {digest: string, reference: string}
```

#### Advanced Tools
```typescript
// Get conversation history
get_agent_session_history(agent_session_id: string, limit?: number) -> {messages: Message[]}

// Execute with custom configuration (requires investigation)
// TODO: Research if/how current cagent supports dynamic agent configuration
// invoke_with_config(config: AgentConfig, message: string) -> {response: string}

// Note: transfer_task is internal to cagent agents and not externally exposed
// Multi-agent delegation happens automatically within agent sessions via transfer_task tool
```

### Session Management

#### Session Lifecycle
1. **Creation**: `create_agent_session()` spawns new cagent runtime
2. **Active**: Multiple `send_message()` calls maintain conversation
3. **Idle**: Session remains in memory but inactive
4. **Cleanup**: Automatic timeout or explicit `close_agent_session()`

#### Session Storage
```go
// Two-tier session architecture
type Manager struct {
    clientSessions  map[string]*ClientSession  // MCP client sessions (automatic)
    mutex          sync.RWMutex
    timeout        time.Duration
    maxSessions    int
}

// ClientSession - Created automatically on MCP client connect
type ClientSession struct {
    ID           string
    agentSessions map[string]*AgentSession    // Agent sessions scoped to this client
    created      time.Time
    lastUsed     time.Time
}

// AgentSession - Created explicitly via create_agent_session tool
type AgentSession struct {
    ID        string
    ClientID  string        // Parent client session ID
    AgentSpec string        // Can be file path, image reference, or store reference
    Runtime   *runtime.Runtime
    Created   time.Time
    LastUsed  time.Time
    Messages  []session.AgentMessage
}
```

### Event Streaming

**MCP supports streaming through Server-Sent Events (SSE) in the HTTP transport**, enabling real-time responses:

#### MCP Streaming Capabilities
- **SSE Support**: HTTP transport supports Server-Sent Events for streaming responses
- **Multiple Messages**: Servers can send multiple JSON-RPC messages before final response  
- **Interactive Experience**: Clients receive real-time updates during agent execution
- **Session Management**: Resumable streams with event IDs for reliability

#### Streaming Implementation Approach
- Use MCP's SSE streaming to forward `runtime.RunStream()` events in real-time
- Each streaming event contains partial response with content, events, and metadata
- Final message indicates completion with full response summary
- Enhanced user experience with immediate feedback during long-running operations

#### Example Streaming Response Flow
```json
// Stream Event 1 - Tool Call Started
{"type": "partial", "content": {"tool_call": "filesystem", "status": "started"}}

// Stream Event 2 - Tool Response  
{"type": "partial", "content": {"tool_response": {"result": "file contents..."}}}

// Stream Event 3 - Agent Message
{"type": "partial", "content": {"agent_message": "Based on the file..."}}

// Final Event - Complete Response
{
  "type": "complete", 
  "response": "final agent response",
  "metadata": {"duration_ms": 1500, "tool_calls": 2, "tokens_used": 450}
}
```

**Selected: SSE Streaming Implementation** to provide optimal interactive experience for MCP clients.

## Code Structure Design

### Architecture Overview

The implementation follows a **layered architecture** with a shared service core that can support multiple transport protocols:

```
┌─────────────────┐    ┌─────────────────┐
│   MCP Server    │    │  HTTP Server    │  
│  (Transport)    │    │  (Transport)    │  <- Future refactor
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          └──────────┬───────────┘
                     │
        ┌─────────────▼─────────────┐
        │     Service Core         │  <- Shared business logic
        │   (Multi-tenant)         │
        └─────────────┬─────────────┘
                      │
        ┌─────────────▼─────────────┐
        │   Existing Components    │
        │  (runtime, loader, etc.) │
        └───────────────────────────┘
```

### Package Organization

#### New Packages to Create

```
pkg/
├── servicecore/                  # Shared multi-tenant service layer
│   ├── manager.go                # Client-scoped session management
│   ├── resolver.go               # Agent resolution (file/image/store)
│   ├── executor.go               # Runtime execution with streaming
│   ├── store.go                  # Multi-tenant session storage
│   └── types.go                  # Common types and interfaces
├── mcpserver/                    # MCP protocol-specific transport
│   ├── server.go                 # MCP server setup and lifecycle
│   ├── handlers.go               # MCP tool implementations using servicecore
│   └── tools.go                  # MCP tool definitions and registration
cmd/root/
└── mcp.go                        # New `cagent mcp run` command
```

#### Future HTTP Server Refactor

```
pkg/
└── server/                       # HTTP transport (Phase 2 refactor)
    ├── server.go                 # HTTP endpoints using servicecore
    ├── handlers.go               # HTTP handlers calling servicecore methods
    └── types.go                  # HTTP-specific request/response types
```

#### Dependencies and Imports

Based on existing codebase analysis, we'll use:
- **MCP Library**: `github.com/mark3labs/mcp-go/server` (confirmed server support)
- **Existing Components**: 
  - `pkg/runtime` - Core agent runtime
  - `pkg/session` - Session message handling (will be wrapped by servicecore)
  - `pkg/loader` - Agent configuration loading
  - `pkg/team` - Agent team management
  - `pkg/content` - Docker image content store
  - `pkg/remote` - Registry operations

### Package Responsibilities

#### `pkg/servicecore/`
**Purpose**: Shared multi-tenant agent service layer

```go
// types.go - Core service interfaces and types
type ServiceManager interface {
    // Client lifecycle
    CreateClient(clientID string) error
    RemoveClient(clientID string) error
    
    // Agent operations
    ResolveAgent(agentSpec string) (string, error)
    ListAgents(source string) ([]AgentInfo, error)
    PullAgent(registryRef string) error
    
    // Session operations (client-scoped)
    CreateAgentSession(clientID, agentSpec string) (*AgentSession, error)
    SendMessage(clientID, sessionID, message string) (*Response, error)
    ListSessions(clientID string) ([]*AgentSession, error)
    CloseSession(clientID, sessionID string) error
}

type AgentInfo struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Source      string `json:"source"` // "file", "store"
    Path        string `json:"path,omitempty"`
    Reference   string `json:"reference,omitempty"`
}

type Response struct {
    Content  string                 `json:"content"`
    Events   []runtime.Event        `json:"events"`
    Metadata map[string]interface{} `json:"metadata"`
}

// manager.go - Client and session management
type Manager struct {
    clients      map[string]*Client
    store        Store
    resolver     *Resolver
    executor     *Executor
    mutex        sync.RWMutex
    logger       *slog.Logger
}

type Client struct {
    ID           string
    AgentSessions map[string]*AgentSession
    Created      time.Time
    LastUsed     time.Time
    mutex        sync.RWMutex
}

type AgentSession struct {
    ID        string
    ClientID  string
    AgentSpec string
    Runtime   *runtime.Runtime
    Session   *session.Session
    Created   time.Time
    LastUsed  time.Time
}

// resolver.go - Agent resolution logic
type Resolver struct {
    agentsDir string
    store     *content.Store
    logger    *slog.Logger
}

func (r *Resolver) ResolveAgent(agentSpec string) (string, error)
func (r *Resolver) ListFileAgents() ([]AgentInfo, error)
func (r *Resolver) ListStoreAgents() ([]AgentInfo, error)
func (r *Resolver) PullAgent(registryRef string) error

// executor.go - Runtime execution with streaming
type Executor struct {
    logger *slog.Logger
}

func (e *Executor) CreateRuntime(agentPath, agentName string, envFiles []string, gateway string) (*runtime.Runtime, error)
func (e *Executor) ExecuteStream(rt *runtime.Runtime, sess *session.Session, message string) (*Response, error)

// store.go - Multi-tenant session storage
type Store interface {
    // Client operations
    CreateClient(ctx context.Context, clientID string) error
    DeleteClient(ctx context.Context, clientID string) error
    
    // Session operations (client-scoped)
    CreateSession(ctx context.Context, clientID string, session *AgentSession) error
    GetSession(ctx context.Context, clientID, sessionID string) (*AgentSession, error)
    ListSessions(ctx context.Context, clientID string) ([]*AgentSession, error)
    UpdateSession(ctx context.Context, clientID string, session *AgentSession) error
    DeleteSession(ctx context.Context, clientID, sessionID string) error
}
```

#### `pkg/mcpserver/`
**Purpose**: MCP protocol transport layer

```go
// server.go
type MCPServer struct {
    server      *server.MCPServer     // from mcp-go
    serviceCore servicecore.ServiceManager
    logger      *slog.Logger
}

func NewMCPServer(serviceCore servicecore.ServiceManager, logger *slog.Logger) *MCPServer
func (s *MCPServer) Start(ctx context.Context) error
func (s *MCPServer) Stop() error

// Client session lifecycle hooks
func (s *MCPServer) OnClientConnect(clientID string) error
func (s *MCPServer) OnClientDisconnect(clientID string) error

// handlers.go - MCP tool implementations using servicecore
func (s *MCPServer) handleCreateAgentSession(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleSendMessage(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleInvokeAgent(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleListAgents(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleCloseAgentSession(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleListAgentSessions(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handlePullAgent(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)

// tools.go - MCP tool registration
func (s *MCPServer) registerTools() error
func createAgentSessionTool() mcp.Tool
func sendMessageTool() mcp.Tool
func invokeAgentTool() mcp.Tool
```

#### Future `pkg/server/` (Phase 2 Refactor)
**Purpose**: HTTP transport layer using servicecore

```go
// server.go - HTTP server using servicecore
type HTTPServer struct {
    e           *echo.Echo
    serviceCore servicecore.ServiceManager
    logger      *slog.Logger
}

// handlers.go - HTTP handlers calling servicecore methods
func (s *HTTPServer) createSession(c echo.Context) error {
    // Extract client ID from HTTP session/auth
    clientID := extractClientID(c)
    
    // Call servicecore instead of direct session creation
    session, err := s.serviceCore.CreateAgentSession(clientID, agentSpec)
    // ...
}

func (s *HTTPServer) listSessions(c echo.Context) error {
    clientID := extractClientID(c)
    sessions, err := s.serviceCore.ListSessions(clientID)  // Client-scoped!
    return c.JSON(http.StatusOK, sessions)
}

// types.go - HTTP-specific request/response types
type CreateSessionRequest struct {
    AgentSpec string `json:"agent_spec"`
}

type SendMessageRequest struct {
    Message string `json:"message"`
}
```

#### `cmd/root/mcp.go`
**Purpose**: CLI command for MCP server mode

```go
func NewMCPCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mcp run",
        Short: "Start cagent in MCP server mode",
        RunE:  runMCPCommand,
    }
    
    cmd.Flags().StringVar(&agentsDir, "agents-dir", "~/.cagent/agents", "Directory containing agent configs")
    cmd.Flags().IntVar(&maxSessions, "max-sessions", 100, "Maximum concurrent sessions")
    cmd.Flags().DurationVar(&sessionTimeout, "session-timeout", time.Hour, "Session timeout duration")
    cmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
    
    return cmd
}

func runMCPCommand(cmd *cobra.Command, args []string) error {
    // Create servicecore manager
    serviceCore := servicecore.NewManager(agentsDir, sessionTimeout, maxSessions, logger)
    
    // Create MCP server using servicecore
    mcpServer := mcpserver.NewMCPServer(serviceCore, logger)
    
    return mcpServer.Start(ctx)
}
```

### Integration Points with Existing Components 

#### Reused Components (via servicecore)
- **`pkg/runtime/runtime.go`**: Core execution engine (reuse `Runtime.RunStream()`)
- **`pkg/session/session.go`**: Message handling and conversation state (wrapped by servicecore)
- **`pkg/loader/loader.go`**: Agent configuration loading 
- **`pkg/team/team.go`**: Agent team management
- **`pkg/tools/`**: All existing tool infrastructure
- **`pkg/content/store.go`**: Docker image content store
- **`pkg/remote/pull.go`**: Registry operations

#### Extended Components
- **`cmd/root/root.go`**: Add MCP command to root command
- **`go.mod`**: Already has `github.com/mark3labs/mcp-go v0.34.0`

#### Security Enhancement & Migration Strategy

**Current State:**
- **`pkg/session/store.go`**: Single-tenant, no client isolation
- **Database schema**: `sessions` table without `client_id` field

**Target State:**
- **`pkg/servicecore/store.go`**: Multi-tenant with client scoping
- **Database schema**: Extended with `client_id` for proper isolation

**Client ID Handling by Transport:**

| Transport | Client ID Source | Strategy |
|-----------|------------------|----------|
| **MCP Server** | MCP session context | Real client ID from MCP library |
| **HTTP API** | Default constant | `__global` until authentication added |
| **Future HTTP + Auth** | Authentication context | Real client ID from auth token/session |

**Non-Breaking Migration Strategy:**
```sql
-- Phase 1: Add client_id column with default
ALTER TABLE sessions ADD COLUMN client_id TEXT DEFAULT '__global';
CREATE INDEX idx_sessions_client_id ON sessions(client_id);

-- Phase 2: All existing sessions automatically get '__global' client_id
-- Phase 3: New sessions get real client_id (MCP) or '__global' (HTTP)
-- Phase 4: When HTTP adds auth, it can use real client_id
```

**Backward Compatibility:**
- ✅ Existing HTTP API clients continue to work unchanged
- ✅ All existing sessions remain accessible via HTTP API
- ✅ HTTP sessions isolated from MCP sessions automatically
- ✅ No data loss or breaking changes

### MCP Tool Definitions

The MCP server will expose these tools to clients:

```go
var mcpTools = []mcp.Tool{
    {
        Name: "create_agent_session",
        Description: "Create a new cagent agent session with specified agent",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]mcp.ToolInputSchemaProperty{
                "agent": {
                    Type: "string", 
                    Description: "Agent specification: file path (e.g., ~/.cagent/agents/echo.yaml), image reference (e.g., docker/echo-agent:latest), or local store reference (e.g., echo-agent:latest)",
                },
                "session_name": {Type: "string", Description: "Optional agent session name"},
            },
            Required: []string{"agent"},
        },
    },
    {
        Name: "invoke_agent",
        Description: "One-shot agent invocation without session persistence",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]mcp.ToolInputSchemaProperty{
                "agent": {
                    Type: "string",
                    Description: "Agent specification: file path, image reference, or local store reference",
                },
                "message": {Type: "string", Description: "Message to send to the agent"},
            },
            Required: []string{"agent", "message"},
        },
    },
    {
        Name: "send_message", 
        Description: "Send message to existing agent session",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]mcp.ToolInputSchemaProperty{
                "agent_session_id": {Type: "string", Description: "Agent session ID (scoped to current client)"},
                "message": {Type: "string", Description: "Message to send"},
            },
            Required: []string{"agent_session_id", "message"},
        },
    },
    {
        Name: "list_agents",
        Description: "List available agents from files and local store",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]mcp.ToolInputSchemaProperty{
                "source": {
                    Type: "string",
                    Description: "Filter by source: 'files' (local YAML files), 'store' (Docker images), or 'all' (default)",
                },
            },
            Required: []string{},
        },
    },
    {
        Name: "list_agent_sessions",
        Description: "List all agent sessions for current client",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]mcp.ToolInputSchemaProperty{},
            Required: []string{},
        },
    },
    {
        Name: "close_agent_session",
        Description: "Close an agent session",
        InputSchema: mcp.ToolInputSchema{
            Type: "object", 
            Properties: map[string]mcp.ToolInputSchemaProperty{
                "agent_session_id": {Type: "string", Description: "Agent session ID to close"},
            },
            Required: []string{"agent_session_id"},
        },
    },
    {
        Name: "pull_agent",
        Description: "Pull agent image from registry to local store",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]mcp.ToolInputSchemaProperty{
                "registry_ref": {Type: "string", Description: "Registry reference (e.g., docker/echo-agent:latest)"},
            },
            Required: []string{"registry_ref"},
        },
    },
}
```

### Integration Flow

#### Client Session Establishment
```
Claude Code → Connect to MCP Server → OnClientConnect(clientID)
                                              ↓
                                 serviceCore.CreateClient(clientID)
                                              ↓
                                    Ready for agent session creation
```

#### Agent Session Creation Flow  
```
Claude Code → create_agent_session → Extract clientID from MCP context
                                              ↓
                                 serviceCore.CreateAgentSession(clientID, agentSpec)
                                              ↓
                                    resolver.ResolveAgent(agentSpec)
                                              ↓
                                        ┌─────────────────┐
                                        │   File Path?    │────Yes───→ Use agentSpec directly
                                        └─────────────────┘
                                              ↓ No
                                    Check content store (fromStore)
                                              ↓
                                    Create temp file with YAML content
                                              ↓  
                                     executor.CreateRuntime(resolved_path)
                                              ↓  
                                     loader.Load() → runtime.New() → session.New()
                                              ↓
                                Store in servicecore with client scoping
```

#### Message Processing Flow (Client-Scoped)
```
Claude Code → send_message → Extract clientID from MCP context
                                      ↓
                    serviceCore.SendMessage(clientID, sessionID, message)
                                      ↓
                             Get client-scoped session from store
                                      ↓ 
                        session.Messages.append(UserMessage)
                                      ↓
                           executor.ExecuteStream(runtime, session, message)
                                      ↓
                         Process events and collect response
                                      ↓
                            Return structured response with metadata
```

### Client Session Isolation

#### Session Scoping Architecture
```
MCP Server
├── ClientSession[claude-code-instance-1]
│   ├── AgentSession[echo-001] 
│   ├── AgentSession[code-reviewer-002]
│   └── AgentSession[finance-003]
├── ClientSession[claude-code-instance-2]  
│   ├── AgentSession[pirate-001]
│   └── AgentSession[writer-002]
└── ClientSession[vscode-extension-1]
    └── AgentSession[debug-helper-001]
```

#### Client ID Extraction
```go
// Extract client ID from MCP session context
func (s *MCPServer) extractClientID(ctx context.Context) (string, error) {
    // Implementation depends on mcp-go library's session context
    // May use connection ID, client info, or custom headers
    if clientSession, ok := ctx.Value("mcp_client_session").(string); ok {
        return clientSession, nil
    }
    return "", fmt.Errorf("no client session found in context")
}

// All tool handlers use client-scoped operations
func (s *MCPServer) handleCreateAgentSession(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error) {
    agentSpec := args["agent"].(string)
    
    // Create agent session scoped to this client
    agentSession, err := s.sessions.CreateAgentSession(clientID, agentSpec)
    if err != nil {
        return nil, err
    }
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            mcp.TextContent{
                Type: "text",
                Text: fmt.Sprintf(`{"agent_session_id": "%s", "agent_spec": "%s"}`, agentSession.ID, agentSpec),
            },
        },
    }, nil
}

// Agent resolution follows the same pattern as run.go
func (s *MCPServer) resolveAgentSource(agentSpec string) (string, error) {
    // Check if it's a file path that exists
    if fileExists(agentSpec) {
        return agentSpec, nil
    }
    
    // Try to load from content store (Docker images)
    store, err := content.NewStore()
    if err != nil {
        return "", fmt.Errorf("creating content store: %w", err)
    }
    
    // Extract YAML content from image (same logic as run.go fromStore)
    img, err := store.GetArtifactImage(agentSpec)
    if err != nil {
        return "", fmt.Errorf("agent not found in files or store: %w", err)
    }
    
    layers, err := img.Layers()
    if err != nil {
        return "", err
    }
    
    var buf bytes.Buffer
    layer := layers[0]
    b, err := layer.Uncompressed()
    if err != nil {
        return "", err
    }
    
    _, err = io.Copy(&buf, b)
    if err != nil {
        return "", err
    }
    b.Close()
    
    // Create temporary file with YAML content
    tmpFile, err := os.CreateTemp("", "mcpagent-*.yaml")
    if err != nil {
        return "", err
    }
    
    if _, err := tmpFile.WriteString(buf.String()); err != nil {
        tmpFile.Close()
        os.Remove(tmpFile.Name())
        return "", err
    }
    
    if err := tmpFile.Close(); err != nil {
        os.Remove(tmpFile.Name())
        return "", err
    }
    
    return tmpFile.Name(), nil
}
```

#### Security Guarantees
- **Agent Session Isolation**: Client A cannot access Client B's agent sessions
- **No Cross-Client Visibility**: `list_agent_sessions` only shows current client's sessions  
- **Scoped Operations**: All operations (send_message, close_session) are client-scoped
- **Automatic Cleanup**: Client disconnect closes all associated agent sessions

### Dependencies Analysis

#### Required Dependencies (Already Available)
- ✅ `github.com/mark3labs/mcp-go v0.34.0` - MCP server support confirmed
- ✅ All existing cagent packages for runtime, session, loader, etc.

#### New Dependencies (None Required)
- All functionality can be built with existing dependencies
- No additional external packages needed

#### Dependency Graph
```
cmd/root/mcp.go
    ↓
pkg/mcpserver/
    ├── pkg/mcpsession/ 
    │   ├── pkg/runtime/
    │   ├── pkg/session/
    │   ├── pkg/team/
    │   └── pkg/loader/
    └── github.com/mark3labs/mcp-go/server
```

## Implementation Plan

### Phase 1: Service Core Foundation
- [ ] Create `pkg/servicecore/` package structure
- [ ] Implement core interfaces and types
- [ ] Multi-tenant session storage with client scoping
- [ ] Agent resolver (file/image/store resolution)
- [ ] Runtime executor with streaming support
- [ ] Service manager with client lifecycle

### Phase 2: MCP Server Implementation
- [ ] Add `cagent mcp run` command
- [ ] Implement MCP server using servicecore
- [ ] MCP tool handlers calling servicecore methods
- [ ] Client identity extraction from MCP context
- [ ] Basic `invoke_agent()` and `list_agents()` tools

### Phase 3: Session Management
- [ ] `create_agent_session()` and `send_message()` tools
- [ ] Client-scoped session operations
- [ ] Session timeout and cleanup
- [ ] Event streaming and response formatting

### Phase 4: Production Features
- [ ] Error handling and recovery
- [ ] Performance optimizations
- [ ] Metrics and monitoring
- [ ] Configuration validation
- [ ] Comprehensive testing

### Phase 5: HTTP Server Refactor (Future)
- [ ] Extract HTTP handlers to use servicecore  
- [ ] Add client authentication to HTTP layer
- [ ] Implement client-scoped operations in HTTP API
- [ ] Migrate existing HTTP clients to secure multi-tenant model

## Agent Specification Formats

The MCP server supports three types of agent specifications:

### 1. File Path Specification
```typescript
// Direct file path to local YAML configuration
const session = await mcp.call("create_agent_session", {
  agent: "~/.cagent/agents/code-reviewer.yaml"
});

// Relative paths also supported
const relativeSession = await mcp.call("create_agent_session", {
  agent: "./agents/echo-agent.yaml"
});
```

### 2. Docker Image Reference
```typescript
// Full registry reference (requires pull_agent first)
const dockerSession = await mcp.call("create_agent_session", {
  agent: "docker/code-reviewer:latest"
});

// Organization/repository format
const orgSession = await mcp.call("create_agent_session", {
  agent: "myorg/custom-agent:v1.2.3"
});
```

### 3. Local Store Reference
```typescript
// Reference to locally stored image (after pull_agent)
const storeSession = await mcp.call("create_agent_session", {
  agent: "code-reviewer:latest"
});

// Works with any tag
const taggedSession = await mcp.call("create_agent_session", {
  agent: "echo-agent:v1.0"
});
```

### Agent Discovery and Management
```typescript
// List agents by source type
const fileAgents = await mcp.call("list_agents", {source: "files"});
const storeAgents = await mcp.call("list_agents", {source: "store"});
const allAgents = await mcp.call("list_agents");  // All sources

// Pull new agent from registry
const pullResult = await mcp.call("pull_agent", {
  registry_ref: "docker/new-agent:latest"
});
// Now available as "new-agent:latest" in store
```

## Usage Examples

### Claude Code Integration
```typescript
// Create agent session for code review agent (automatically scoped to this client)
// File path approach
const session = await mcp.call("create_agent_session", {
  agent: "~/.cagent/agents/code-reviewer.yaml"
});

// Docker image approach
const dockerSession = await mcp.call("create_agent_session", {
  agent: "docker/code-reviewer:latest"
});

// Local store approach
const storeSession = await mcp.call("create_agent_session", {
  agent: "code-reviewer:latest"
});

// Send code for review
const review = await mcp.call("send_message", {
  agent_session_id: session.agent_session_id,
  message: "Please review this Python function: def calculate_tax(income): return income * 0.25"
});

// Follow up questions in same context
const followup = await mcp.call("send_message", {
  agent_session_id: session.agent_session_id, 
  message: "What about edge cases for negative income?"
});

// List all agent sessions for this client
const mySessions = await mcp.call("list_agent_sessions");

// Close specific agent session
await mcp.call("close_agent_session", {
  agent_session_id: session.agent_session_id
});
```

### Programmatic Agent Invocation
```typescript
// One-shot invocation (no session persistence)
// File path approach
const result = await mcp.call("invoke_agent", {
  agent: "~/.cagent/agents/echo-agent.yaml",
  message: "Hello world"
});

// Docker image approach
const dockerResult = await mcp.call("invoke_agent", {
  agent: "docker/echo-agent:latest",
  message: "Hello world"
});

// Local store approach
const storeResult = await mcp.call("invoke_agent", {
  agent: "echo-agent:latest",
  message: "Hello world"
});

// List available agents with source filtering
const allAgents = await mcp.call("list_agents");  // All sources
const fileAgents = await mcp.call("list_agents", {source: "files"});  // File-based only
const storeAgents = await mcp.call("list_agents", {source: "store"});  // Docker images only

// Pull agent from registry to local store
const pullResult = await mcp.call("pull_agent", {
  registry_ref: "docker/code-reviewer:latest"
});
```

### Multi-Client Isolation Example
```typescript
// Client A (Claude Code instance 1)
const sessionA = await mcp.call("create_agent_session", {
  agent: "code-reviewer:latest"  // Store reference
}); // Creates agent_session_id: "a1b2c3"

// Client B (Claude Code instance 2) 
const sessionB = await mcp.call("create_agent_session", {
  agent: "code-reviewer:latest"  // Same store reference
}); // Creates agent_session_id: "d4e5f6" (different ID, isolated)

// Client A cannot access Client B's session
await mcp.call("send_message", {
  agent_session_id: "d4e5f6",  // Client B's session ID
  message: "Hello"
}); // ERROR: Agent session not found (properly isolated)
```

## Technical Considerations

### Performance
- Agent loading optimization (caching configurations)
- Memory management for long-running sessions
- Concurrent session handling
- Connection pooling for external tools

### Security
- File path validation for agent configurations
- Session isolation (prevent cross-session access)
- Resource limits per session
- Input sanitization

### Reliability
- Graceful error handling and recovery
- Session state persistence (optional)
- Health checks and monitoring
- Automatic session cleanup

### Scalability
- Maximum concurrent sessions
- Session eviction policies (LRU, timeout-based)
- Resource monitoring and limits
- Horizontal scaling considerations

## Integration Points

### Existing cagent Components
- **Runtime System**: Reuse existing `runtime.Runtime`
- **Session Management**: Extend `pkg/session`
- **Configuration**: Leverage `pkg/config` and `pkg/loader`
- **Tool System**: Utilize existing tool infrastructure

### External Dependencies
- MCP protocol implementation
- HTTP/WebSocket server (for MCP transport)
- Session storage (in-memory + optional persistence)

## Benefits

1. **Programmatic Access**: External tools can invoke cagent without CLI
2. **Rich Integration**: Claude Code can use cagent agents as sophisticated tools
3. **Session Continuity**: Multi-turn conversations with context preservation
4. **Scalability**: Multiple concurrent agent sessions
5. **Flexibility**: Both one-shot and conversational modes supported

## Testing Strategy

### MCP Server Testing Capabilities

The `github.com/mark3labs/mcp-go/server` package provides built-in testing facilities:

#### Available Test Functions
```go
// For SSE transport testing
func NewTestServer(server *MCPServer, opts ...SSEOption) *httptest.Server

// For streamable HTTP transport testing  
func NewTestStreamableHTTPServer(server *MCPServer, opts ...StreamableHTTPOption) *httptest.Server
```

These create in-memory HTTP test servers using Go's `httptest.Server`, avoiding inter-process communication.

### Testing Approach

#### Unit Tests Structure
```
pkg/mcpserver/
├── server_test.go           # MCPServer lifecycle tests
├── handlers_test.go         # Individual tool handler tests
├── sessions_test.go         # Session management tests
└── integration_test.go      # End-to-end MCP workflow tests

pkg/mcpsession/
├── manager_test.go          # Session manager tests
├── session_test.go          # Individual session tests
└── store_test.go           # Persistence tests
```

#### Test Implementation Pattern

**ServiceCore Testing with Isolation:**
```go
func TestManager_AgentOperations(t *testing.T) {
    // Create isolated store for testing
    store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
    require.NoError(t, err)

    resolver, err := NewResolverWithStore(tempDir, store, logger)
    require.NoError(t, err)

    manager, err := NewManagerWithResolver(resolver, time.Hour, 10, logger)
    require.NoError(t, err)

    // Test with isolated store ensures no production data contamination
    storeAgents, err := manager.ListAgents("store")
    require.NoError(t, err)
    assert.Len(t, storeAgents, 0) // Predictable results with clean store
}
```

**MCP Server Testing:**
```go
func TestMCPServer_CreateSession(t *testing.T) {
    // Setup
    logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
    mcpServer := NewMCPServer("testdata/agents", logger)
    
    // Create test server (in-memory)
    testServer := server.NewTestServer(mcpServer.server)
    defer testServer.Close()
    
    // Create test client
    client, err := client.NewStreamableHttpClient(testServer.URL)
    require.NoError(t, err)
    
    // Test create_agent_session tool
    result, err := client.CallTool(ctx, mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name: "create_agent_session",
            Arguments: map[string]any{
                "agent": "testdata/agents/echo-agent.yaml",
            },
        },
    })
    
    // Assertions
    require.NoError(t, err)
    assert.Contains(t, result.Content, "agent_session_id")
}
```

#### Integration Test Strategy
```go
func TestMCPServer_FullWorkflow(t *testing.T) {
    // Test complete workflow:
    // 1. Create session
    // 2. Send multiple messages  
    // 3. Verify conversation state
    // 4. Close session
    // 5. Verify cleanup
}
```

#### Mock Agent Configurations
```
testdata/
├── agents/
│   ├── echo-agent.yaml          # Simple echo for basic tests
│   ├── complex-agent.yaml       # Multi-tool agent for advanced tests
│   └── invalid-agent.yaml       # For error handling tests
└── configs/
    └── test-models.yaml         # Test model configurations
```

### Testing Benefits

#### Test Isolation Advantages
- **Clean environments** - Each test uses isolated temporary stores preventing cross-test contamination
- **Predictable results** - Tests don't depend on external state or leftover artifacts from other tests
- **Production safety** - Tests never modify user's real content store (`~/.cagent/store/`)
- **Dependency injection** - Constructors like `NewManagerWithResolver()` enable custom store configurations
- **Parallel execution** - Tests can run concurrently without interfering with each other

#### In-Memory Testing Advantages
- **Fast execution** - No process spawning or network overhead
- **Deterministic** - Controlled environment without external dependencies
- **Easy debugging** - All code runs in same process with full debugging
- **CI/CD friendly** - No complex setup or cleanup required

#### Test Coverage Areas
1. **Tool handlers** - Individual MCP tool functionality
2. **Client session isolation** - Cross-client security and scoping
3. **Agent session management** - Concurrent access, timeouts, cleanup
4. **Agent loading** - Configuration parsing and validation
5. **Error handling** - Invalid inputs, missing files, runtime errors
6. **Integration** - Full client→server→agent→response workflows
7. **Test isolation** - Dependency injection patterns with temporary stores

#### Client Session Isolation Tests
```go
func TestClientSessionIsolation(t *testing.T) {
    // Setup server with multiple test clients
    mcpServer := NewMCPServer("testdata/agents", logger)
    testServer := server.NewTestServer(mcpServer.server)
    defer testServer.Close()
    
    // Create two separate clients
    clientA, err := client.NewStreamableHttpClient(testServer.URL)
    require.NoError(t, err)
    
    clientB, err := client.NewStreamableHttpClient(testServer.URL) 
    require.NoError(t, err)
    
    // Client A creates agent session
    sessionA, err := clientA.CallTool(ctx, mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name: "create_agent_session",
            Arguments: map[string]any{"agent": "testdata/agents/echo-agent.yaml"},
        },
    })
    require.NoError(t, err)
    
    // Client B creates agent session  
    sessionB, err := clientB.CallTool(ctx, mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name: "create_agent_session", 
            Arguments: map[string]any{"agent": "testdata/agents/echo-agent.yaml"},
        },
    })
    require.NoError(t, err)
    
    // Extract session IDs
    var sessionAData, sessionBData map[string]string
    json.Unmarshal([]byte(sessionA.Content[0].(mcp.TextContent).Text), &sessionAData)
    json.Unmarshal([]byte(sessionB.Content[0].(mcp.TextContent).Text), &sessionBData)
    
    // Client A should not be able to access Client B's session
    _, err = clientA.CallTool(ctx, mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name: "send_message",
            Arguments: map[string]any{
                "agent_session_id": sessionBData["agent_session_id"],
                "message": "Hello",
            },
        },
    })
    assert.Error(t, err, "Client A should not access Client B's session")
    
    // Client A should be able to access its own session
    _, err = clientA.CallTool(ctx, mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name: "send_message",
            Arguments: map[string]any{
                "agent_session_id": sessionAData["agent_session_id"],
                "message": "Hello",
            },
        },
    })
    assert.NoError(t, err, "Client A should access its own session")
}
```

### Alternative Testing Options

#### Option 1: Direct Handler Testing (Recommended)
Test MCP tool handlers directly without MCP protocol overhead:
```go
func TestHandleCreateAgentSession_Direct(t *testing.T) {
    server := NewMCPServer("testdata/agents", logger)
    
    result, err := server.handleCreateAgentSession(ctx, "client-1", map[string]any{
        "agent": "testdata/agents/echo-agent.yaml",
    })
    
    // Test result directly
}
```

#### Option 2: Mock MCP Client 
Create test doubles for external dependencies:
```go
type MockMCPClient struct {
    calls []ToolCall
}

func (m *MockMCPClient) CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Record calls and return test responses
}
```

This design provides a robust foundation for exposing cagent as a programmable service while maintaining its existing capabilities and architecture.