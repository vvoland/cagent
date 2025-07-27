# MCP Testing Examples

This directory contains tools and examples for testing cagent's MCP server functionality.

## test-mcp-client.go

A comprehensive MCP client for testing the cagent MCP server's agent management functionality.

### Features

- Tests store agent listing before and after pulls
- Verifies file agent discovery
- Tests agent pulling from Docker registries
- Validates combined agent listing (files + store)
- Proper MCP handshake and SSE transport

### Usage

1. **Start the MCP server:**
   ```bash
   ./bin/cagent mcp run --port 8080 --path /mcp --debug --agents-dir examples/config
   ```

2. **Run the test client:**
   ```bash
   cd examples/mcptesting
   go run test-mcp-client.go
   ```

3. **Expected output:**
   - Initial store listing (shows existing pulled agents)
   - Pull operation success
   - Updated store listing (same or new agents depending on cache)
   - Combined listing showing both file and store agents

### Environment Variables

- `MCP_SSE_ENDPOINT`: Override the default endpoint (default: `http://localhost:8080/mcp/sse`)

### Test Scenarios

The client tests the following workflow:

1. **Store listing (before)**: Lists agents currently in the content store
2. **Agent pull**: Pulls `djordjelukic1639080/jean-laurent` from registry
3. **Store listing (after)**: Verifies store contents after pull
4. **Combined listing**: Shows all available agents (files + store)

### Key Validations

- ✅ Store agents are properly listed from content store
- ✅ File agents are discovered from agents directory
- ✅ Pull operations complete successfully
- ✅ Agent name extraction works correctly for Docker references
- ✅ MCP protocol handshake and tool calls function properly

## Adding New Tests

To add new MCP test scenarios:

1. Create new test functions following the pattern in `test-mcp-client.go`
2. Use the same MCP client initialization and tool call patterns
3. Add proper error handling and result validation
4. Document expected behavior in this README