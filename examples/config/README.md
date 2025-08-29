# YAML Configuration Files Analysis

## **Basic Configurations:**

| Name          | Description/Purpose                                 | Filesystem | Shell | Todo | Think | Memory | MCP Servers | Sub-agents |
|---------------|-----------------------------------------------------|------------|-------|------|-------|--------|-------------|------------|
| 42.yaml       | Douglas Adams-style witty AI assistant              | ❌         | ❌    | ❌   | ❌    | ❌     | ❌          | ❌         |
| pythonist.yaml| Python programming assistant                        | ✅         | ✅    | ❌   | ❌    | ❌     | ❌          | ❌         |
| script_shell.yaml | Agent with custom shell commands                | ❌         | ✅    | ❌   | ❌    | ❌     | ❌          | ❌         |
| echo-agent.yaml | Simple echo agent                                 | ❌         | ❌    | ❌   | ❌    | ❌     | ❌          | ❌         |
| contradict.yaml | Contrarian viewpoint provider                     | ❌         | ❌    | ❌   | ❌    | ❌     | ❌          | ❌         |
| github.yaml   | Github assistance using MCP tools                   | ❌         | ❌    | ❌   | ❌    | ❌     | `github-official` | ❌     |
| mem.yaml      | Humorous AI with persistent memory                  | ✅         | ❌    | ❌   | ❌    | ✅     | ❌          | ❌         |
| airbnb.yaml   | Airbnb search specialist                            | ❌         | ❌    | ❌   | ❌    | ❌     | `@openbnb/mcp-server-airbnb` | ❌ |
| diag.yaml     | Log analysis and diagnostics                        | ✅         | ✅    | ❌   | ✅    | ❌     | ❌          | ❌         |
| pirate.yaml   | Pirate-themed assistant                             | ❌         | ❌    | ❌   | ❌    | ❌     | ❌          | ❌         |

## **Advanced Configurations:**

| Name                       | Description/Purpose                           | Filesystem | Shell | Todo | Think | Memory | MCP Servers  | Sub-agents |
|----------------------------|-----------------------------------------------|------------|-------|------|-------|--------|--------------|------------|
| bio.yaml                   | Biography generation from internet searches   | ❌         | ❌    | ❌   | ❌    | ❌     | `duckduckgo, fetch` | ❌     |
| github_issue_manager.yaml  | GitHub Issue Manager                          | ❌         | ❌    | ❌   | ❌    | ❌     | `github`          | ❌     |
| alloy.yaml                 | Learning assistant                            | ❌         | ❌    | ❌   | ❌    | ❌     | ❌                | ❌     |
| review.yaml                | Dockerfile review specialist                  | ✅         | ❌    | ❌   | ❌    | ❌     | ❌                | ❌     |
| code.yaml                  | Code analysis and development assistant       | ✅         | ✅    | ✅   | ❌    | ❌     | ❌                | ❌     |
| go_packages.yml            | Golang packages expert                        | ❌         | ❌    | ❌   | ❌    | ❌     | ❌                | ❌     |
| silvia.yaml                | Sylvia Plath-inspired poetic AI               | ❌         | ❌    | ❌   | ❌    | ❌     | ❌                | ❌     |
| todo.yaml                  | Task manager example                          | ❌         | ❌    | ✅   | ❌    | ❌     | ❌                | ❌     |
| image_text_extractor.yaml  | Image text extraction                         | ✅         | ❌    | ❌   | ❌    | ❌     | ❌                | ❌     |
| doc_generator.yaml         | Documentation generation from codebases       | ❌         | ✅    | ❌   | ✅    | ❌     | ❌                | ❌     |

## **Multi-Agent Configurations:**

| Name          | Description/Purpose                        | Filesystem | Shell | Todo | Think | Memory | MCP Servers  | Sub-agents     |
|---------------|--------------------------------------------|------------|-------|------|-------|--------|--------------|----------------|
| agent.yaml    | Docker Expert Assistant                    | ❌         | ❌    | ❌   | ❌    | ❌     | ❌            | ✅             |
| blog.yaml     | Technical blog writing workflow            | ❌         | ❌    | ❌   | ✅    | ❌     | `duckduckgo-mcp-server` | ✅   |
| dev-team.yaml | Development team coordinator               | ✅         | ✅    | ✅   | ✅    | ✅     | ❌            | ✅             |
| multi-code.yaml | Technical lead and project coordination  | ✅         | ✅    | ✅   | ✅    | ✅     | ❌            | ✅             |
| writer.yaml   | Story writing workflow supervisor          | ❌         | ❌    | ❌   | ✅    | ❌     | ❌            | ✅             |
| finance.yaml  | Financial research and analysis            | ❌         | ❌    | ❌   | ✅    | ❌     | `duckduckgo-mcp-server` | ✅ |
| shared-todo.yaml | Shared todo item manager                | ❌         | ❌    | ✅   | ❌    | ❌     | ❌            | ✅             |
| mcp_generator.yaml | Generates MCP configurations         | ❌         | ❌    | ❌   | ❌    | ❌     | `docker,duckduckgo-mcp-server` | ❌ |
| moby.yaml     | Moby Project Expert                        | ❌         | ❌    | ❌   | ❌    | ❌     | `gitmcp.io/moby/moby` | ❌ |
| dmr.yaml      | Pirate-themed AI assistant                 | ❌         | ❌    | ❌   | ❌    | ❌     | ❌            | ❌             |
