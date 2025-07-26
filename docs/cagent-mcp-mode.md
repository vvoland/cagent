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
create_session(agent_file: string, session_name?: string) -> {session_id: string}

// Send message to existing session  
send_message(session_id: string, message: string) -> {response: string, events: Event[]}

// One-shot invocation (stateless)
invoke_agent(agent_file: string, message: string) -> {response: string}

// List available agents
list_agents(directory?: string) -> {agents: AgentInfo[]}

// Session management
get_session_info(session_id: string) -> {agent: string, message_count: number, created: string}
list_sessions() -> {sessions: SessionInfo[]}
close_session(session_id: string) -> {success: boolean}
```

#### Advanced Tools
```typescript
// Get conversation history
get_session_history(session_id: string, limit?: number) -> {messages: Message[]}

// Execute with custom configuration
invoke_with_config(config: AgentConfig, message: string) -> {response: string}

// Multi-agent delegation within session
transfer_session(session_id: string, target_agent: string) -> {success: boolean}
```

### Session Management

#### Session Lifecycle
1. **Creation**: `create_session()` spawns new cagent runtime
2. **Active**: Multiple `send_message()` calls maintain conversation
3. **Idle**: Session remains in memory but inactive
4. **Cleanup**: Automatic timeout or explicit `close_session()`

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

## Implementation Plan

### Phase 1: Basic MCP Server
- [ ] Add `cagent mcp run` command
- [ ] Implement MCP server infrastructure
- [ ] Basic `invoke_agent()` tool (one-shot mode)
- [ ] `list_agents()` functionality

### Phase 2: Session Management
- [ ] Session manager implementation
- [ ] `create_session()` and `send_message()` tools
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

## Usage Examples

### Claude Code Integration
```typescript
// Create session for code review agent
const session = await mcp.call("create_session", {
  agent_file: "~/.automation/cagent/agents/code-reviewer.yaml"
});

// Send code for review
const review = await mcp.call("send_message", {
  session_id: session.session_id,
  message: "Please review this Python function: def calculate_tax(income): return income * 0.25"
});

// Follow up questions in same context
const followup = await mcp.call("send_message", {
  session_id: session.session_id, 
  message: "What about edge cases for negative income?"
});
```

### Programmatic Agent Invocation
```typescript
// One-shot invocation
const result = await mcp.call("invoke_agent", {
  agent_file: "~/.automation/cagent/agents/echo-agent.yaml",
  message: "Hello world"
});

// List available agents
const agents = await mcp.call("list_agents");
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

This design provides a robust foundation for exposing cagent as a programmable service while maintaining its existing capabilities and architecture.