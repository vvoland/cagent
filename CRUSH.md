# CRUSH.md - Development Guide for cagent

## Build & Test Commands

- `task build` - Build application binary
- `task test` - Run all Go tests
- `go test ./pkg/servicecore` - Run tests for specific package
- `go test -run TestStoreBasicOperations ./pkg/content` - Run single test
- `task lint` - Run golangci-lint with gocritic, revive rules
- `task link` - Create symlink to ~/bin for easy access

## Code Style Guidelines

### Imports & Formatting

- Use `goimports` for import organization (standard, third-party, local)
- Group imports: stdlib, external, internal (`github.com/docker/cagent/pkg/...`)
- Follow golangci-lint rules: gocritic, revive with exported comments required
- Get rid of unwanted trailing spaces in yaml files

### Types & Naming

- Use descriptive struct names: `Agent`, `ServiceManager`, `Toolset`
- Private fields use camelCase: `toolsetsStarted`, `memoryManager`
- Public methods use PascalCase: `Name()`, `Instruction()`
- Interface names end with -er when possible: `Provider`, `Manager`

### Error Handling

- Wrap errors with context: `fmt.Errorf("creating resolver: %w", err)`
- Use structured logging with slog: `slog.Error("message", "key", value)`
- Validate inputs early with descriptive errors
- Use atomic operations for concurrent access: `atomic.Bool`

### Architecture Patterns

- Multi-tenant design: all operations require clientID scoping
- Use dependency injection via functional options: `New(name, prompt, ...Opt)`
- Implement proper interfaces: `ServiceManager`, `Provider`, `ToolSet`
- Thread-safe operations with proper mutex usage: `sync.RWMutex`
- Comprehensive package documentation explaining purpose and security considerations
