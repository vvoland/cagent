# A2A Integration

This document describes how to expose Docker `cagent` agents via Google's A2A (Agent-to-Agent) protocol.

## Features

- Auto-selects available port if not specified
- Loads agents from files or agent catalog
- Supports all Docker `cagent` features (tools, models gateway, etc.)
- Provides agent metadata via standard A2A agent card

## Usage

```bash
# Start A2A server for an agent
cagent a2a ./agent.yaml

# Specify a custom port
cagent a2a ./agent.yaml --port 8080

# Use an agent from the catalog
cagent a2a agentcatalog/pirate --port 9000
```

## Limitations and Future Work

1. **Tool Call Visibility**: Currently, tool calls are handled internally and not exposed as separate ADK events
2. **Artifacts**: ADK artifact support not yet integrated
3. **Memory**: ADK memory features not yet integrated
4. **Sub-agents**: Cagent teams/multi-agent scenarios need further work
5. **Callbacks**: ADK before/after agent callbacks not yet used