# YAML Configuration Files Analysis

## **Basic Configurations:**


| Name              | Description/Purpose                                 | Filesystem | Shell | Todo | Think | Memory | MCP Servers | Sub-agents |
|------------------ |-----------------------------------------------------|------------|-------|------|-------|--------|-------------|------------|
| echo-agent.yaml   | Simple echo agent                                   |            |       |      |       |        |             |           |
| pirate.yaml       | Pirate-themed assistant                             |            |       |      |       |        |             |           |
| 42.yaml           | Douglas Adams-style witty AI assistant              |            |       |      |       |        |             |           |
| contradict.yaml   | Contrarian viewpoint provider                       |            |       |      |       |        |             |           |
| silvia.yaml       | Sylvia Plath-inspired poetic AI                     |            |       |      |       |        |             |           |
| script_shell.yaml | Agent with custom shell commands                    |            | ✓     |      |       |        |             |           |
| mem.yaml          | Humorous AI with persistent memory                  | ✓          |       |      |       | ✓      |             |           |
| diag.yaml         | Log analysis and diagnostics                        | ✓          | ✓     |      | ✓     |        |             |           |
| todo.yaml         | Task manager example                                |            |       | ✓    |       |        |             |           |
| pythonist.yaml    | Python programming assistant                        | ✓          | ✓     |      |       |        |             |           |
| alloy.yaml        | Learning assistant                                  |            |       |      |       |        |             |           |
| dmr.yaml          | Pirate-themed AI assistant                          |            |       |      |       |        |             |           |


## **Advanced Configurations:**

| Name                       | Description/Purpose                          | Filesystem | Shell | Todo | Think | Memory | MCP Servers  | Sub-agents |
|----------------------------|----------------------------------------------|------------|-------|------|-------|--------|--------------|------------|
| bio.yaml                   | Biography generation from internet searches  |            |       |      |       |        | `duckduckgo, fetch` |       |
| airbnb.yaml                | Airbnb search specialist                     |            |       |      |       |        | `@openbnb/mcp-server-airbnb` |   |
| github_issue_manager.yaml  | GitHub Issue Manager                         |            |       |      |       |        | `github-official`          |       |
| github.yaml                | Github assistance using MCP tools            |            |       |      |       |        | `github-official` |    |
| review.yaml                | Dockerfile review specialist                 | ✓          |       |      |       |        |              |       |
| code.yaml                  | Code analysis and development assistant      | ✓          | ✓     | ✓    |       |        |              |       |
| go_packages.yml            | Golang packages expert                       |            |       |      |       |        |              |       |
| moby.yaml                  | Moby Project Expert                          |            |       |      |       |        | `gitmcp.io/moby/moby` |   |
| image_text_extractor.yaml  | Image text extraction                        | ✓          |       |      |       |        |              |       |
| doc_generator.yaml         | Documentation generation from codebases      |            | ✓     |      | ✓     |        |              |       |
| mcp_generator.yaml         | Generates MCP configurations                 |            |       |      |       |        | `docker,duckduckgo-mcp-server` |   |

## **Multi-Agent Configurations:**

| Name              | Description/Purpose                        | Filesystem | Shell | Todo | Think | Memory | MCP Servers  | Sub-agents     |
|-------------------|--------------------------------------------|------------|-------|------|-------|--------|--------------|----------------|
| agent.yaml        | Docker Expert Assistant                    |            |       |      |       |        |              | ✓             |
| blog.yaml         | Technical blog writing workflow            |            |       |      | ✓     |        | `duckduckgo-mcp-server` | ✓   |
| dev-team.yaml     | Development team coordinator               | ✓          | ✓     | ✓    | ✓     | ✓      |              | ✓             |
| multi-code.yaml   | Technical lead and project coordination    | ✓          | ✓     | ✓    | ✓     | ✓      |              | ✓             |
| writer.yaml       | Story writing workflow supervisor          |            |       |      | ✓     |        |              | ✓             |
| finance.yaml      | Financial research and analysis            |            |       |      | ✓     |        | `duckduckgo-mcp-server` | ✓ |
| shared-todo.yaml  | Shared todo item manager                   |            |       | ✓    |       |        |              | ✓             |

