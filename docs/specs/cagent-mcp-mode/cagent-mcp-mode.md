# cagent MCP Server Mode Design

## Overview

This document outlines the design for an MCP (Model Context Protocol) server mode for cagent, allowing external clients like Claude Code to programmatically invoke cagent agents and maintain conversational sessions.

## Architecture Options

### Option 1: One-Shot Mode (Stateless)
Each MCP call creates a fresh cagent runtime instance:
- Load agent configuration
- Process single message
- Return response
- Terminate runtime

**Pros:**
- Simple implementation
- No session management complexity  
- Stateless and predictable
- No memory accumulation

**Cons:**
- High overhead (loading agents repeatedly)
- No conversation context between calls
- Cannot leverage agent memory features
- Inefficient for multi-turn interactions

### Option 2: Session-Based Mode (Stateful) - **RECOMMENDED**
MCP server maintains persistent cagent sessions:
- Sessions map to long-running cagent runtime instances
- Conversation state preserved across calls
- Multi-turn interactions supported
- Session lifecycle management

**Pros:**
- Efficient resource usage
- True conversational flow
- Supports agent memory and context
- Enables complex multi-turn workflows

**Cons:**
- Complex session management
- Memory usage grows over time
- Need session cleanup/timeout logic

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

// Execute with custom configuration
invoke_with_config(config: AgentConfig, message: string) -> {response: string}

// Multi-agent delegation within session
transfer_agent_session(agent_session_id: string, target_agent: string) -> {success: boolean}
```

### Session Management

#### Session Lifecycle
1. **Creation**: `create_agent_session()` spawns new cagent runtime
2. **Active**: Multiple `send_message()` calls maintain conversation
3. **Idle**: Session remains in memory but inactive
4. **Cleanup**: Automatic timeout or explicit `close_agent_session()`

#### Session Storage
```go
type Session struct {
    ID        string
    AgentFile string
    Runtime   *runtime.Runtime
    Created   time.Time
    LastUsed  time.Time
    Messages  []session.AgentMessage
}

type SessionManager struct {
    sessions    map[string]*Session
    mutex       sync.RWMutex
    timeout     time.Duration
    maxSessions int
}
```

### Event Streaming

Since MCP doesn't natively support streaming, we'll need to handle cagent's event-driven responses:

#### Option A: Buffered Response
- Collect all events from `runtime.RunStream()`
- Return final response + event summary
- Simple but loses real-time feedback

#### Option B: Event Array Response
- Return structured response with event details
- Client can process events for rich feedback
- More complex but preserves information

```json
{
  "response": "final agent response",
  "events": [
    {"type": "tool_call", "tool": "filesystem", "args": "..."},
    {"type": "tool_response", "result": "..."},
    {"type": "agent_message", "content": "..."}
  ],
  "metadata": {
    "duration_ms": 1500,
    "tool_calls": 2,
    "tokens_used": 450
  }
}
```

## Code Structure Design

### Package Organization

#### New Packages to Create

```
pkg/
├── mcpserver/                    # MCP server implementation
│   ├── server.go                 # Core MCP server setup and lifecycle
│   ├── handlers.go               # MCP tool handlers (create_session, send_message, etc.)
│   ├── sessions.go               # Session management logic
│   └── tools.go                  # MCP tool definitions and registration
├── mcpsession/                   # Session management
│   ├── manager.go                # SessionManager with concurrent access
│   ├── session.go                # Session wrapper around runtime.Runtime
│   └── store.go                  # Optional persistence for sessions
cmd/root/
└── mcp.go                        # New `cagent mcp run` command
```

#### Dependencies and Imports

Based on existing codebase analysis, we'll use:
- **MCP Library**: `github.com/mark3labs/mcp-go/server` (confirmed server support)
- **Existing Components**: 
  - `pkg/runtime` - Core agent runtime
  - `pkg/session` - Session message handling
  - `pkg/loader` - Agent configuration loading
  - `pkg/team` - Agent team management

### Package Responsibilities

#### `pkg/mcpserver/`
**Purpose**: Core MCP server implementation and tool handlers

```go
// server.go
type MCPServer struct {
    server     *server.MCPServer  // from mcp-go
    sessions   *mcpsession.Manager
    logger     *slog.Logger
    agentsDir  string
}

func NewMCPServer(agentsDir string, logger *slog.Logger) *MCPServer
func (s *MCPServer) Start(ctx context.Context) error
func (s *MCPServer) Stop() error

// handlers.go - MCP tool implementations (client-scoped)
func (s *MCPServer) handleCreateAgentSession(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleSendMessage(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleInvokeAgent(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleListAgents(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleCloseAgentSession(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handleListAgentSessions(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)
func (s *MCPServer) handlePullAgent(ctx context.Context, clientID string, args map[string]any) (*mcp.CallToolResult, error)

// Agent resolution utilities
func (s *MCPServer) resolveAgentSource(agentSpec string) (string, error)
func (s *MCPServer) listFileAgents() ([]AgentInfo, error)
func (s *MCPServer) listStoreAgents() ([]AgentInfo, error)

// Client session lifecycle hooks
func (s *MCPServer) OnClientConnect(clientID string) error
func (s *MCPServer) OnClientDisconnect(clientID string) error

// tools.go - MCP tool registration
func (s *MCPServer) registerTools() error
func createSessionTool() mcp.Tool
func sendMessageTool() mcp.Tool
func invokeAgentTool() mcp.Tool
```

#### `pkg/mcpsession/`
**Purpose**: Client-scoped session lifecycle and state management

```go
// manager.go
type Manager struct {
    clientSessions  map[string]*ClientSession  // MCP client sessions
    mutex          sync.RWMutex
    timeout        time.Duration
    maxSessions    int
    logger         *slog.Logger
}

// ClientSession represents an MCP client connection (e.g., one Claude Code instance)
type ClientSession struct {
    ID           string
    agentSessions map[string]*AgentSession    // Agent sessions scoped to this client
    mutex        sync.RWMutex
    created      time.Time
    lastUsed     time.Time
}

// AgentSession represents a running cagent instance within a client session
type AgentSession struct {
    ID        string
    ClientID  string        // Parent client session ID
    AgentSpec string        // Can be file path, image reference, or store reference
    Runtime   *runtime.Runtime
    Team      *team.Team
    Session   *session.Session
    Created   time.Time
    LastUsed  time.Time
    mutex     sync.Mutex
}

func NewManager(timeout time.Duration, maxSessions int, logger *slog.Logger) *Manager
func (m *Manager) GetOrCreateClientSession(clientID string) *ClientSession
func (m *Manager) CreateAgentSession(clientID, agentSpec string) (*AgentSession, error)
func (m *Manager) GetAgentSession(clientID, agentSessionID string) (*AgentSession, error)
func (m *Manager) CloseAgentSession(clientID, agentSessionID string) error
func (m *Manager) CloseClientSession(clientID string) error
func (m *Manager) CleanupExpired() int

// session.go  
func (s *AgentSession) SendMessage(message string) (*Response, error)
func (s *AgentSession) GetHistory() []session.AgentMessage
func (s *AgentSession) Close() error

func (cs *ClientSession) CreateAgentSession(agentSpec string) (*AgentSession, error)
func (cs *ClientSession) GetAgentSession(agentSessionID string) (*AgentSession, error)
func (cs *ClientSession) ListAgentSessions() []*AgentSession
func (cs *ClientSession) CloseAgentSession(agentSessionID string) error
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

func runMCPCommand(cmd *cobra.Command, args []string) error
```

### Integration Points with Existing Components 

#### Reused Components
- **`pkg/runtime/runtime.go`**: Core execution engine (reuse `Runtime.RunStream()`)
- **`pkg/session/session.go`**: Message handling and conversation state
- **`pkg/loader/loader.go`**: Agent configuration loading 
- **`pkg/team/team.go`**: Agent team management
- **`pkg/tools/`**: All existing tool infrastructure

#### Extended Components
- **`cmd/root/root.go`**: Add MCP command to root command
- **`go.mod`**: Already has `github.com/mark3labs/mcp-go v0.34.0`

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
                                 Create ClientSession in Manager
                                              ↓
                                    Ready for agent session creation
```

#### Agent Session Creation Flow  
```
Claude Code → create_agent_session → Extract clientID from MCP context
                                              ↓
                                 Manager.CreateAgentSession(clientID, agentSpec)
                                              ↓
                                    resolveAgentSource(agentSpec)
                                              ↓
                                        ┌─────────────────┐
                                        │   File Path?    │────Yes───→ Use agentSpec directly
                                        └─────────────────┘
                                              ↓ No
                                    Check content store (fromStore)
                                              ↓
                                    Create temp file with YAML content
                                              ↓  
                                     loader.Load(resolved_path)
                                              ↓  
                                     runtime.New(team, agent)
                                              ↓
                                     session.New(logger)
                                              ↓
                                Store in ClientSession.agentSessions[agentSessionID]
```

#### Message Processing Flow (Client-Scoped)
```
Claude Code → send_message → Extract clientID from MCP context
                                      ↓
                    Manager.GetAgentSession(clientID, agentSessionID)
                                      ↓
                             AgentSession.SendMessage(message)
                                      ↓ 
                        session.Messages.append(UserMessage)
                                      ↓
                           runtime.RunStream(ctx, session)
                                      ↓
                         Process events and collect response
                                      ↓
                            Return structured response
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

### Phase 1: Basic MCP Server
- [ ] Add `cagent mcp run` command
- [ ] Implement MCP server infrastructure
- [ ] Basic `invoke_agent()` tool (one-shot mode)
- [ ] `list_agents()` functionality

### Phase 2: Session Management
- [ ] Session manager implementation
- [ ] `create_agent_session()` and `send_message()` tools
- [ ] Session timeout and cleanup
- [ ] Basic session persistence

### Phase 3: Advanced Features
- [ ] Event streaming/buffering
- [ ] Session history and metadata
- [ ] Multi-agent transfers within sessions
- [ ] Performance optimizations

### Phase 4: Production Features
- [ ] Authentication/authorization
- [ ] Rate limiting
- [ ] Metrics and monitoring
- [ ] Configuration validation
- [ ] Error handling improvements

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
    
    // Test create_session tool
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