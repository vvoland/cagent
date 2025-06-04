# How-to Guide: Practical Agent Configuration

This guide provides step-by-step instructions for common agent configuration
scenarios. Each example includes working code you can copy and adapt for your
needs.

## Prerequisites

- Basic understanding of YAML syntax
- cagent installed and working
- API keys for your chosen AI provider (OpenAI, Anthropic, etc.)
- Docker installed (for MCP tools that require it)

## 🚀 Quick Start: Your First Agent

### Basic Assistant Agent

Create `basic-assistant.yaml`:

```yaml
agents:
  root:
    name: helpful_assistant
    model: gpt4
    description: A knowledgeable assistant for general questions
    instruction: |
      You are a helpful, knowledgeable assistant. Always provide accurate,
      well-structured responses. If you're unsure about something, say so
      rather than guessing.

models:
  gpt4:
    type: openai
    model: gpt-4o
    temperature: 0.7
    max_tokens: 2000
```

**Run it:**

```bash
./cagent -config basic-assistant.yaml
```

### Professional Specialist Agent

Create `financial-advisor.yaml`:

```yaml
agents:
  root:
    name: financial_advisor
    model: claude_smart
    description: Professional financial advisory agent
    instruction: |
      You are a certified financial advisor with expertise in:
      - Investment planning and portfolio management
      - Risk assessment and diversification strategies  
      - Tax-efficient investment approaches

      **Your Process:**
      1. Always ask about the client's risk tolerance and time horizon first
      2. Provide multiple investment options with pros/cons
      3. Explain your reasoning clearly
      4. Include appropriate disclaimers about financial advice

      **Constraints:**
      - Never guarantee specific returns
      - Always recommend diversified approaches
      - Suggest consulting with a human advisor for major decisions

models:
  claude_smart:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.3
    max_tokens: 3000
```

## 🔧 Adding Tools to Agents

### Research Agent with Web Search

Create `research-agent.yaml`:

```yaml
agents:
  root:
    name: research_specialist
    model: claude_research
    description: Research agent with web search capabilities
    instruction: |
      You are a professional research specialist. When users ask questions
      that require current information, use web search to find accurate data.

      **Research Process:**
      1. Understand the research question
      2. Use web search to gather current information  
      3. Analyze multiple sources for accuracy
      4. Synthesize findings into a clear summary
      5. Always cite your sources

      **Source Quality:**
      - Prefer authoritative sources (.edu, .gov, established publications)
      - Cross-reference information across multiple sources
      - Note the recency of information when relevant
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
    think: true

models:
  claude_research:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.4
    max_tokens: 4000
```

### File Management Agent

Create `file-manager.yaml`:

```yaml
agents:
  root:
    name: file_manager
    model: gpt4_files
    description: Agent for file operations and document management
    instruction: |
      You are a file management specialist. You can help users:
      - Read and analyze file contents
      - Create and modify documents
      - Organize file structures
      - Search through file contents

      **Safety First:**
      - Always confirm destructive operations before executing
      - Back up important files when making major changes
      - Respect file permissions and security
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-filesystem"]

models:
  gpt4_files:
    type: openai
    model: gpt-4o
    temperature: 0.2
    max_tokens: 2500
```

### Multi-Tool Power Agent

Create `power-agent.yaml`:

```yaml
agents:
  root:
    name: power_assistant
    model: claude_power
    description: Multi-tool agent for complex tasks
    instruction: |
      You are a power user assistant with access to multiple tools.
      Use the appropriate tool for each task:

      **Web Search**: For current information, news, research
      **File Operations**: For document creation, editing, analysis  
      **Database Access**: For structured data queries and analysis

      **Workflow:**
      1. Analyze the user's request
      2. Determine which tools are needed
      3. Execute tasks using appropriate tools
      4. Synthesize results into a comprehensive response
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-filesystem"]
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-sqlite"]
    think: true

models:
  claude_power:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.5
    max_tokens: 4000
```

## 🎭 Multi-Agent Teams

### Software Development Team

Create `dev-team.yaml`:

```yaml
agents:
  root:
    name: dev_lead
    model: gpt4_lead
    description: Development team coordinator
    instruction: |
      You are a senior development team lead coordinating a software project.

      **Your Responsibilities:**
      - Understand project requirements and break them into tasks
      - Delegate to appropriate specialists based on the task type
      - Review and integrate work from team members
      - Ensure code quality and best practices

      **Delegation Strategy:**
      - Architecture & design questions → architect
      - Code implementation → developer  
      - Code review & quality → reviewer
      - Testing & QA → tester

      **Communication Style:**
      - Be clear and decisive in delegation
      - Provide context when assigning tasks
      - Synthesize specialist input into actionable recommendations
    sub_agents: [architect, developer, reviewer, tester]

  architect:
    name: system_architect
    model: claude_architect
    description: System architecture and design specialist
    instruction: |
      You are a senior software architect specializing in system design.

      **Expertise Areas:**
      - Scalable system architecture patterns
      - Database design and optimization
      - API design and microservices
      - Performance and security considerations

      **Deliverables:**
      - System architecture diagrams and explanations
      - Database schemas and relationships
      - API specifications and design decisions
      - Technology stack recommendations with rationale

  developer:
    name: senior_developer
    model: gpt4_dev
    description: Senior software engineer for implementation
    instruction: |
      You are a senior software developer focused on implementation.

      **Skills:**
      - Full-stack development (frontend/backend/database)
      - Clean code principles and design patterns
      - Performance optimization
      - Security best practices

      **Code Standards:**
      - Write readable, maintainable code
      - Include comprehensive error handling
      - Add clear documentation and comments
      - Follow established coding conventions

  reviewer:
    name: code_reviewer
    model: claude_reviewer
    description: Code review and quality assurance specialist
    instruction: |
      You are a code review specialist focused on quality assurance.

      **Review Areas:**
      - Code correctness and logic
      - Security vulnerabilities
      - Performance issues
      - Maintainability and readability
      - Test coverage and quality

      **Review Process:**
      1. Analyze code for logical errors and edge cases
      2. Check for security vulnerabilities
      3. Evaluate performance implications
      4. Assess code maintainability
      5. Provide specific, actionable feedback

  tester:
    name: qa_specialist
    model: gpt4_qa
    description: Testing and quality assurance specialist
    instruction: |
      You are a QA specialist focused on comprehensive testing.

      **Testing Types:**
      - Unit testing strategies and implementation
      - Integration testing approaches
      - End-to-end testing scenarios
      - Performance and load testing

      **Deliverables:**
      - Test plans and test cases
      - Automated test implementations
      - Bug reports with reproduction steps
      - Testing best practices recommendations

models:
  gpt4_lead:
    type: openai
    model: gpt-4o
    temperature: 0.4
    max_tokens: 3000

  claude_architect:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.3
    max_tokens: 4000

  gpt4_dev:
    type: openai
    model: gpt-4o
    temperature: 0.2
    max_tokens: 3000

  claude_reviewer:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.2
    max_tokens: 3000

  gpt4_qa:
    type: openai
    model: gpt-4o
    temperature: 0.3
    max_tokens: 2500
```

### Content Creation Team

Create `content-team.yaml`:

```yaml
agents:
  root:
    name: content_director
    model: claude_director
    description: Content strategy and production coordinator
    instruction: |
      You are a content director managing a content creation team.

      **Your Role:**
      - Understand content objectives and target audience
      - Develop content strategy and messaging
      - Delegate to specialists based on content type and needs
      - Ensure brand consistency and quality standards

      **Team Specializations:**
      - Research and fact-checking → researcher
      - Writing and copywriting → writer
      - Content editing and polish → editor
      - SEO and content optimization → seo_specialist
    sub_agents: [researcher, writer, editor, seo_specialist]
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]

  researcher:
    name: content_researcher
    model: claude_research
    description: Content research and fact-checking specialist
    instruction: |
      You are a content researcher specializing in thorough fact-checking.

      **Research Process:**
      1. Identify key claims and facts that need verification
      2. Search for authoritative sources and current information
      3. Cross-reference multiple sources for accuracy
      4. Document sources and provide citations
      5. Flag any claims that cannot be verified

      **Source Standards:**
      - Prefer authoritative sources (.edu, .gov, established publications)
      - Ensure information recency for time-sensitive topics
      - Verify statistical claims and data
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]

  writer:
    name: content_writer
    model: gpt4_writer
    description: Professional content writer and copywriter
    instruction: |
      You are a professional content writer with expertise in various formats.

      **Writing Specialties:**
      - Blog posts and articles
      - Marketing copy and sales content
      - Technical documentation
      - Social media content

      **Writing Standards:**
      - Clear, engaging, and audience-appropriate tone
      - Strong headlines and compelling introductions
      - Logical structure with smooth transitions
      - Call-to-action when appropriate

  editor:
    name: content_editor
    model: claude_editor
    description: Content editing and quality specialist
    instruction: |
      You are a content editor focused on polish and quality.

      **Editing Focus:**
      - Grammar, spelling, and punctuation
      - Clarity and readability improvements
      - Tone and voice consistency
      - Structure and flow optimization
      - Fact accuracy and citation verification

      **Quality Standards:**
      - Error-free grammar and spelling
      - Consistent style and formatting
      - Engaging and scannable content
      - Proper attribution and citations

  seo_specialist:
    name: seo_optimizer
    model: gpt4_seo
    description: SEO and content optimization specialist
    instruction: |
      You are an SEO specialist focused on content optimization.

      **Optimization Areas:**
      - Keyword research and integration
      - Meta titles and descriptions
      - Header structure (H1, H2, H3)
      - Internal and external linking strategies
      - Content structure for featured snippets

      **SEO Best Practices:**
      - Natural keyword integration (avoid keyword stuffing)
      - Mobile-friendly content structure
      - Fast-loading content considerations
      - User intent optimization

models:
  claude_director:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.5
    max_tokens: 3000

  claude_research:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.3
    max_tokens: 4000

  gpt4_writer:
    type: openai
    model: gpt-4o
    temperature: 0.7
    max_tokens: 3000

  claude_editor:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.2
    max_tokens: 2500

  gpt4_seo:
    type: openai
    model: gpt-4o
    temperature: 0.3
    max_tokens: 2000
```

## 💭 Advanced Features

### Using the Think Tool

The think tool enables metacognitive reasoning for complex problems.

Create `thinking-agent.yaml`:

```yaml
agents:
  root:
    name: strategic_thinker
    model: claude_think
    description: Strategic planning agent with advanced reasoning
    instruction: |
      You are a strategic planning consultant who uses structured thinking
      to solve complex business problems.

      **Thinking Process:**
      1. Use the think tool to break down complex problems
      2. Consider multiple perspectives and potential solutions
      3. Evaluate pros/cons of different approaches
      4. Validate your reasoning before presenting recommendations

      **When to Think:**
      - Multi-step strategic planning problems
      - Complex decision-making scenarios  
      - When analyzing trade-offs and risks
      - Before making important recommendations

      **Thinking Structure:**
      - Problem definition and scope
      - Key stakeholders and constraints
      - Potential solutions and alternatives
      - Risk assessment and mitigation
      - Implementation considerations
    think: true

models:
  claude_think:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.4
    max_tokens: 4000
```

### Agent with Date Context

Some agents benefit from knowing the current date for time-sensitive responses.

```yaml
agents:
  root:
    name: news_analyst
    model: gpt4_news
    description: Current events and news analysis agent
    instruction: |
      You are a news analyst who provides current event analysis and context.
      Always consider the current date when discussing events and their relevance.
    add_date: true
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]

models:
  gpt4_news:
    type: openai
    model: gpt-4o
    temperature: 0.6
    max_tokens: 3000
```

## 🔄 Common Patterns & Best Practices

### Pattern 1: Specialist Router

Route requests to the most appropriate specialist:

```yaml
agents:
  root:
    name: request_router
    model: gpt4_router
    description: Intelligent request routing coordinator
    instruction: |
      You are a request routing specialist. Analyze incoming requests and
      delegate to the most appropriate specialist:

      **Routing Rules:**
      - Technical/code questions → tech_specialist
      - Business/strategy questions → business_specialist  
      - Creative/content questions → creative_specialist
      - Research questions → research_specialist

      **Always explain your routing decision briefly.**
    sub_agents:
      [
        tech_specialist,
        business_specialist,
        creative_specialist,
        research_specialist,
      ]

  tech_specialist:
    name: tech_specialist
    instruction: "You are a senior technical consultant specializing in software engineering and technology solutions."
    # ... detailed tech instructions

  # ... other specialists
```

### Pattern 2: Sequential Processing Pipeline

Process complex tasks through a multi-stage pipeline:

```yaml
agents:
  root:
    name: content_pipeline
    instruction: |
      You coordinate a content creation pipeline:
      1. research_agent gathers information
      2. writer_agent creates initial content  
      3. editor_agent refines and polishes
      4. seo_agent optimizes for search

      Pass work between agents in sequence, ensuring each stage is complete.
    sub_agents: [research_agent, writer_agent, editor_agent, seo_agent]
```

### Pattern 3: Parallel Consultation

Get input from multiple specialists simultaneously:

```yaml
agents:
  root:
    instruction: |
      For investment decisions, consult ALL specialists in parallel:
      - risk_analyst evaluates potential risks
      - market_analyst assesses market conditions
      - financial_analyst reviews financial metrics

      Synthesize their input into balanced recommendations.
    sub_agents: [risk_analyst, market_analyst, financial_analyst]
```

## 🛠️ Model Configuration Tips

### Temperature Guidelines

```yaml
models:
  creative_model: # High creativity
    temperature: 0.8

  balanced_model: # Balanced creativity/consistency
    temperature: 0.5

  analytical_model: # Low creativity, high consistency
    temperature: 0.2

  factual_model: # Minimal creativity, maximum consistency
    temperature: 0.1
```

### Token Limits by Use Case

```yaml
models:
  brief_responses: # Chat, quick answers
    max_tokens: 1000

  standard_responses: # General purpose
    max_tokens: 2000

  detailed_analysis: # In-depth analysis
    max_tokens: 4000

  comprehensive_docs: # Documentation, reports
    max_tokens: 8000
```

## 🐛 Troubleshooting Common Issues

### Issue: Agent Not Using Tools

**Problem**: Agent has tools configured but doesn't use them **Solution**:
Update instructions to explicitly mention tools

```yaml
instruction: |
  You have access to web search. ALWAYS search for current information
  before answering questions about recent events or data.

  **Available Tools:**
  - search(query): Search the web for current information

  **When to Search:**
  - Any question about events after 2024
  - Statistical data or current numbers
  - Recent news or developments
```

### Issue: Poor Delegation Decisions

**Problem**: Root agent doesn't delegate appropriately **Solution**: Provide
clear delegation guidelines

```yaml
instruction: |
  **Delegation Decision Tree:**

  Is this a coding question? → delegate to developer
  Is this about system design? → delegate to architect
  Is this about testing? → delegate to qa_specialist

  **Always delegate rather than attempting complex tasks yourself.**
```

### Issue: Inconsistent Responses

**Problem**: Agent responses vary too much **Solution**: Lower temperature and
add more specific constraints

```yaml
models:
  consistent_model:
    temperature: 0.2 # Lower for consistency

agents:
  root:
    instruction: |
      **Response Format:**
      Always structure responses as:
      1. Brief summary (1-2 sentences)
      2. Detailed explanation
      3. Actionable next steps

      **Tone:** Professional, helpful, concise
```

## 🎯 Quick Reference

### Essential Configuration Template

```yaml
agents:
  root:
    name: agent_name
    model: model_reference
    description: Brief agent description
    instruction: |
      Role definition and behavioral guidelines

      **Workflow:**
      1. Step-by-step process

      **Tools:** (if applicable)
      - Tool descriptions

      **Constraints:**
      - Behavioral limits and requirements
    tools: [] # Optional
    sub_agents: [] # Optional
    think: false # Optional
    add_date: false # Optional

models:
  model_reference:
    type: openai # or anthropic
    model: gpt-4o # or claude-3-5-sonnet-latest
    temperature: 0.7 # 0.0-1.0
    max_tokens: 2000 # Integer
```

### Model Selection Guide

**OpenAI Models:**

- `gpt-4o`: Best for general intelligence and coding
- `gpt-4-turbo`: Fast, capable alternative
- `gpt-3.5-turbo`: Budget option for simple tasks

**Anthropic Models:**

- `claude-3-5-sonnet-latest`: Best for reasoning and analysis
- `claude-3-haiku`: Fast, budget-friendly option

### Common Tool Commands

```yaml
# Web search
- type: mcp
  command: npx
  args: ["-y", "@modelcontextprotocol/server-brave-search"]

# File operations
- type: mcp
  command: npx
  args: ["-y", "@modelcontextprotocol/server-filesystem"]

# Database access
- type: mcp
  command: npx
  args: ["-y", "@modelcontextprotocol/server-sqlite"]

# Custom Docker tool
- type: mcp
  command: docker
  args: ["run", "-i", "--rm", "your-tool-image"]
```

## 🚀 Next Steps

1. **Start Simple**: Begin with a basic single-agent configuration
2. **Add Tools**: Extend capabilities with MCP tools as needed
3. **Create Specialists**: Add sub-agents for specialized tasks
4. **Optimize**: Tune temperature and token limits for your use case
5. **Scale**: Build complex multi-agent teams for sophisticated workflows

## 📚 Additional Resources

- **[Tutorial](./tutorial.md)** - Step-by-step first agent guide
- **[Explanation](./explanation.md)** - Concepts and architecture deep-dive
- **[Reference](./reference.md)** - Complete configuration documentation
- **[Examples](../examples/)** - Ready-to-use configuration templates

## 💡 Pro Tips

1. **Clear Instructions**: Specific instructions produce better results than
   vague ones
2. **Tool Documentation**: Always explain available tools in agent instructions
3. **Delegation Rules**: Provide clear guidelines for when to use sub-agents
4. **Temperature Tuning**: Lower values for consistency, higher for creativity
5. **Iterative Testing**: Test and refine configurations based on real usage
