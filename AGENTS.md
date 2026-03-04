## Development Commands

### Build and Development

- `task build` - Build the application binary (outputs to `./bin/docker-agent`)
- `task test` - Run Go tests (clears API keys to ensure deterministic tests)
- `task lint` - Run golangci-lint (uses `.golangci.yml` configuration)
- `task format` - Format code using golangci-lint fmt
- `task dev` - Run lint, test, and build in sequence

### Docker and Cross-Platform Builds

- `task build-local` - Build binary for local platform using Docker Buildx
- `task cross` - Build binaries for multiple platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64)
- `task build-image` - Build Docker image tagged as `docker/docker-agent`
- `task push-image` - Build and push multi-platform Docker image to registry

### Running docker-agent

- `./bin/docker-agent run <config.yaml>` - Run agent with configuration (launches TUI by default)
- `./bin/docker-agent run <config.yaml> -a <agent_name>` - Run specific agent from multi-agent config
- `./bin/docker-agent run agentcatalog/pirate` - Run agent directly from OCI registry
- `./bin/docker-agent run --exec <config.yaml>` - Execute agent without TUI (non-interactive)
- `./bin/docker-agent new` - Generate new agent configuration interactively
- `./bin/docker-agent new --model openai/gpt-5` - Generate with specific model
- `./bin/docker-agent share push ./agent.yaml namespace/repo` - Push agent to OCI registry
- `./bin/docker-agent share pull namespace/repo` - Pull agent from OCI registry
- `./bin/docker agent serve mcp ./agent.yaml` - Expose agents as MCP tools
- `./bin/docker agent serve a2a <config.yaml>` - Start agent as A2A server
- `./bin/docker agent serve api` - Start Docker `docker-agent` API server

### Debug and Development Flags

- `--debug` or `-d` - Enable debug logging (logs to `~/.cagent/cagent.debug.log`)
- `--log-file <path>` - Specify custom debug log location
- `--otel` or `-o` - Enable OpenTelemetry tracing
- Example: `./bin/docker-agent run config.yaml --debug --log-file ./debug.log`

### Single Test Execution

- `go test ./pkg/specific/package` - Run tests for specific package
- `go test ./pkg/... -run TestSpecificFunction` - Run specific test function
- `go test -v ./...` - Run all tests with verbose output
- `go test -parallel 1 ./...` - Run tests serially (useful for debugging)

## Development Guidelines

### Testing

- Tests located alongside source files (`*_test.go`)
- Run `task test` to execute full test suite
- E2E tests in `e2e/` directory
- Test fixtures and data in `testdata/` subdirectories