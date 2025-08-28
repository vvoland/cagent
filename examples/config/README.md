1. BASIC AGENTS (Single agent, minimal toolsets, simple functionality)
1.1 Style & Personality Demos
42.yaml: Douglas Adams-style witty assistant (Hitchhiker's Guide persona)
contradict.yaml: Contrarian viewpoint provider
pirate.yaml: Simple pirate-speaking agent
silvia.yaml: Sylvia Plath-inspired poetic AI
dmr.yaml: Basic pirate agent with custom DMR/Qwen model
1.2 Simple Utility Agents
echo-agent.yaml: Minimal echo agent that repeats user input exactly
mem.yaml: Basic memory demonstration with persistent storage
alloy.yaml: Learning companion with multiple model support (claude, gpt-4o)
1.3 Basic Development Tools
review.yaml: Docker file reviewer (filesystem toolset)
todo.yaml: Code editor with filesystem and todo tools
pythonist.yaml: Python development assistant (filesystem, shell)
2. INTERMEDIATE AGENTS (Single agent with multiple toolsets or moderate complexity)
2.1 Web-Integrated Agents
airbnb.yaml: Travel accommodation search with AirBnB MCP server
bio.yaml: Biography generator using DuckDuckGo and fetch tools
moby.yaml: Moby project expert with remote MCP integration
github.yaml: GitHub assistant using official GitHub MCP server
2.2 Specialized Development Tools
go_packages.yml: Go package expert with custom script toolset
image_text_extractor.yaml: OCR and image analysis with vision models
diag.yaml: Log analysis specialist (filesystem, shell, think)
2.3 Custom Script Agents
script_shell.yaml: Demonstrates custom shell commands (IP, Docker, GitHub APIs)
3. ADVANCED SINGLE AGENTS (Complex workflows, multiple toolsets, sophisticated functionality)
3.1 Code Generation & Analysis
code.yaml: Expert code analysis with validation loops (filesystem, shell, todo, DuckDuckGo)
mcp_generator.yaml: Python code generator with testing workflow (think, DuckDuckGo, filesystem, shell)
doc_generator.yaml: Comprehensive documentation generator (shell, think, Diátaxis framework)
3.2 Specialized Migration Tools
dhi/dhi.yaml: Docker Hardened Images migration specialist (DuckDuckGo, filesystem, shell, todo)
3.3 Project Management
github_issue_manager.yaml: GitHub issue management with date awareness
4. MULTI-AGENT SYSTEMS (Coordinated teams with specialized sub-agents)
4.1 Content Creation Teams
blog.yaml: Technical blog writing (web_search_agent + writer)
writer.yaml: Creative writing workflow (prompt_chooser + writer)
finance.yaml: Financial analysis (finance_agent + web_search_agent)
4.2 Development Teams
agent.yaml: Docker expertise team (containerize + optimize_dockerfile + pirate)
multi-code.yaml: Full development team (web frontend + golang backend specialist)
dev-team.yaml: Complete development workflow (designer + awesome_engineer with memory)
4.3 Language Processing Teams
professional/professional_writing_agent.yaml: English editing + French translation
4.4 Shared State Demos
shared-todo.yaml: Demonstrates shared todo state between agents
5. EVALUATION & TESTING
eval/agent.yaml: Basic evaluation agent with shell access
eval/README.md: Framework for agent evaluation with scoring metrics
TOOLSET ANALYSIS
Built-in Tools Usage:
filesystem: 15 agents (most common for file operations)
shell: 10 agents (execution and validation)
think: 8 agents (reasoning and planning)
todo: 6 agents (task management)
memory: 3 agents (persistent state)
script: 2 agents (custom commands)
MCP Server Integration:
DuckDuckGo: 6 agents (web search capabilities)
GitHub: 3 agents (repository integration)
Financial tools: 2 agents (yfmcp for Yahoo Finance)
Specialized: AirBnB, fetch, shell-server
Model Diversity:
Anthropic Claude: Most popular (20+ configurations)
OpenAI GPT-4o: Second most common (15+ configurations)
Custom models: DMR/Qwen, Gemini integration examples
Multi-model: Several agents support multiple model providers
KEY INSIGHTS
Complexity Spectrum: From simple personality demos to sophisticated multi-agent development teams
Tool Integration: Heavy emphasis on filesystem operations and shell execution for practical tasks
MCP Ecosystem: Strong integration with Model Context Protocol servers for extended capabilities
Development Focus: Many examples target software development workflows (Docker, code analysis, documentation)
Collaborative Patterns: Advanced examples show sophisticated agent coordination and task delegation
Real-world Applications: Examples cover practical use cases like financial analysis, content creation, and project management
This comprehensive collection demonstrates the full range of capabilities from simple chatbots to complex autonomous development teams.
