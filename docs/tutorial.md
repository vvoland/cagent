# Step-by-Step Tutorial: Build Your First Agent

Welcome to the complete cagent tutorial! This hands-on guide will take you from zero to a working multi-agent system. You'll build increasingly sophisticated agents and learn best practices along the way.

## 🎯 What You'll Build

By the end of this tutorial, you'll have created:

1. **Basic Assistant** - Simple conversational agent
2. **Research Agent** - Agent with web search capabilities
3. **Multi-Agent Team** - Coordinated specialists working together
4. **Advanced System** - Full-featured agent with thinking capabilities

## 📋 Prerequisites

Before starting, ensure you have:

- **cagent installed** and working on your system
- **API keys** for OpenAI and/or Anthropic
- **Basic YAML knowledge** (we'll explain as we go)
- **Text editor** (VS Code, Sublime, etc.)
- **Terminal/command line** access

## 🛠️ Setup

### 1. Verify Installation

```bash
# Check that cagent is installed
./cagent --help

# You should see the help output
```

### 2. Set API Keys

```bash
# For OpenAI
export OPENAI_API_KEY=sk-your-api-key-here

# For Anthropic (optional)
export ANTHROPIC_API_KEY=sk-ant-your-api-key-here
```

### 3. Create Tutorial Directory

```bash
mkdir cagent-tutorial
cd cagent-tutorial
```

## 🚀 Chapter 1: Your First Agent

Let's start with the simplest possible agent to understand the basics.

### Step 1: Create a Basic Agent

Create a file called `basic-agent.yaml`:

```yaml
agents:
  root:
    name: my_first_agent
    model: gpt4
    description: A friendly AI assistant
    instruction: |
      You are a friendly and helpful AI assistant. Always:
      - Be polite and professional
      - Provide clear, concise answers
      - Ask clarifying questions when needed

models:
  gpt4:
    type: openai
    model: gpt-4o
    temperature: 0.7
    max_tokens: 2000
```

### Step 2: Test Your First Agent

```bash
# Run your agent
./cagent -config basic-agent.yaml

# Try asking questions like:
# "Hello, what can you help me with?"
# "What's the weather like today?"
# "Explain quantum computing in simple terms"
```

### Step 3: Understanding the Configuration

Let's break down what each part does:

```yaml
agents:
  root: # Required: main agent entry point
    name: my_first_agent # Agent identifier (3-50 characters)
    model: gpt4 # References the model below
    description: "..." # Brief description (max 200 chars)
    instruction: | # Multi-line instructions (most important!)
      You are a friendly...

models:
  gpt4: # Model identifier (referenced above)
    type: openai # Provider: "openai" or "anthropic"
    model: gpt-4o # Specific model variant
    temperature: 0.7 # Creativity level (0.0-1.0)
    max_tokens: 2000 # Maximum response length
```

**🎉 Congratulations!** You've created your first agent. Try different questions and see how it responds.

## 🔧 Chapter 2: Adding Tools

Now let's give your agent superpowers by adding toolsets. Toolsets let agents interact with the outside world.

### Step 1: Create a Research Agent

Create `research-agent.yaml`:

```yaml
agents:
  root:
    name: research_assistant
    model: claude_research
    description: AI research assistant with web search
    instruction: |
      You are a professional research assistant with access to web search.

      **Your Process:**
      1. Listen carefully to research requests
      2. Use web search to find current, accurate information
      3. Analyze multiple sources for reliability
      4. Provide well-sourced, comprehensive answers
      5. Always cite your sources

      **Source Quality Guidelines:**
      - Prefer authoritative sources (.edu, .gov, established news)
      - Look for recent information when relevance matters
      - Cross-check facts across multiple sources

      **Available Tools:**
      - search(query): Search the web for current information
      - summarize(text): Summarize long text passages
    toolsets:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
        tools: ["search", "summarize"] # Only enable these specific tools

models:
  claude_research:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.4
    max_tokens: 3000
```

### Step 2: Test Your Research Agent

```bash
# Run the research agent
./cagent -config research-agent.yaml

# Try research questions:
# "What are the latest developments in AI in 2024?"
# "Find information about sustainable energy trends"
# "Research the current stock market performance"
```

### Step 3: Understanding Toolsets

Toolsets extend what agents can do:

- **Web Search**: Gets current information from the internet
- **File Operations**: Reads and writes files
- **Database Access**: Queries databases
- **Custom Toolsets**: Your own specialized tools

Each toolset can expose multiple tools, and you can optionally filter which tools are available to the agent using the `tools` field.

Notice how we:

1. **Documented the available tools** in the instruction
2. **Explained when to use them** (for current information)
3. **Set quality standards** for source evaluation
4. **Filtered specific tools** we want to enable

**💡 Pro Tip**: Always tell your agent about its available tools in the instructions!

## 👥 Chapter 3: Building a Multi-Agent Team

The real power of cagent comes from agents working together. Let's build a team!

### Step 1: Create a Simple Team

Create `simple-team.yaml`:

```yaml
agents:
  root:
    name: team_coordinator
    model: gpt4_coordinator
    description: Coordinates a team of AI specialists
    instruction: |
      You are a team coordinator managing AI specialists. Your job is to:

      **Route tasks to the right specialist:**
      - Writing tasks → content_writer
      - Research questions → researcher
      - Technical questions → tech_expert

      **Always explain your routing decision** to help users understand
      why you're delegating to a specific team member.

      **Team Communication:**
      - Introduce the specialist you're calling
      - Provide clear context about the task
      - Synthesize their response if needed
    sub_agents: [content_writer, researcher, tech_expert]

  content_writer:
    name: content_writer
    model: gpt4_creative
    description: Professional content writer and copywriter
    instruction: |
      You are a professional content writer specializing in:
      - Blog posts and articles
      - Marketing copy
      - Social media content
      - Creative writing

      **Writing Style:**
      - Engaging and conversational tone
      - Clear structure with good flow
      - Audience-appropriate language
      - Strong headlines and conclusions

  researcher:
    name: researcher
    model: claude_analytical
    description: Research specialist with analytical focus
    instruction: |
      You are a research specialist who excels at:
      - Finding and analyzing information
      - Fact-checking and verification
      - Summarizing complex topics
      - Providing evidence-based conclusions

      **Research Standards:**
      - Always cite sources when possible
      - Present balanced viewpoints
      - Distinguish between facts and opinions
      - Acknowledge limitations in available data

  tech_expert:
    name: tech_expert
    model: claude_technical
    description: Technical specialist for engineering questions
    instruction: |
      You are a technical expert specializing in:
      - Software engineering and development
      - System architecture and design
      - Programming languages and frameworks
      - Technical problem-solving

      **Technical Communication:**
      - Explain complex concepts clearly
      - Provide practical examples and code when helpful
      - Consider different skill levels in explanations
      - Focus on actionable solutions

models:
  gpt4_coordinator:
    type: openai
    model: gpt-4o
    temperature: 0.5
    max_tokens: 2000

  gpt4_creative:
    type: openai
    model: gpt-4o
    temperature: 0.8
    max_tokens: 3000

  claude_analytical:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.3
    max_tokens: 3000

  claude_technical:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.2
    max_tokens: 3000
```

### Step 2: Test Your Team

```bash
# Run the team
./cagent -config simple-team.yaml

# Try different types of requests:
# "Write a blog post about sustainable living"
# "Research the benefits of meditation"
# "Explain how to build a REST API in Python"
# "Create a social media post about productivity"
```

### Step 3: Understanding Multi-Agent Systems

Notice what happens:

1. **Coordinator analyzes** your request
2. **Routes to specialist** based on task type
3. **Specialist handles** the specific work
4. **Response flows back** through coordinator

**Key Benefits:**

- **Specialization**: Each agent excels in specific domains
- **Modularity**: Easy to add/remove/modify specialists
- **Scalability**: Can handle complex, multi-step workflows

**🎯 Observe**: Pay attention to how the coordinator explains its routing decisions!

## 🧠 Chapter 4: Advanced Features

Let's explore advanced features that make agents even more powerful.

### Step 1: Add the Think Tool

Create `thinking-agent.yaml`:

```yaml
agents:
  root:
    name: strategic_thinker
    model: claude_thinking
    description: Strategic planning agent with advanced reasoning
    instruction: |
      You are a strategic planning consultant who thinks through complex problems.

      **Your Superpower: The Think Tool**
      Before responding to complex questions, use the think tool to:
      1. Break down the problem into components
      2. Consider multiple approaches and perspectives
      3. Evaluate pros and cons of different solutions
      4. Plan your response structure

      **When to Think:**
      - Multi-step strategic problems
      - Questions with trade-offs or competing priorities
      - Complex analysis requiring structured reasoning
      - Before making important recommendations

      **Thinking Process:**
      - Problem definition: What exactly is being asked?
      - Context analysis: What factors are relevant?
      - Option generation: What are possible approaches?
      - Evaluation: What are the pros/cons of each?
      - Synthesis: What's the best path forward?
    think: true
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]

models:
  claude_thinking:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.4
    max_tokens: 4000
```

### Step 2: Test Advanced Reasoning

```bash
# Run the thinking agent
./cagent -config thinking-agent.yaml

# Try complex problems:
# "Help me decide between starting a business or staying in my job"
# "What's the best strategy for reducing carbon emissions in cities?"
# "How should a company approach digital transformation?"
```

### Step 3: Combine Everything

Create `ultimate-agent.yaml` - our most advanced system:

```yaml
agents:
  root:
    name: ultimate_assistant
    model: claude_coordinator
    description: Advanced AI system with thinking, tools, and specialists
    instruction: |
      You are an advanced AI assistant with a complete toolkit:

      **Your Capabilities:**
      - Web search for current information
      - Advanced reasoning with the think tool
      - Team of specialists for complex tasks

      **Decision Framework:**
      1. For simple questions: Answer directly
      2. For current info needs: Use web search
      3. For complex analysis: Use think tool first
      4. For specialized tasks: Delegate to team members

      **Team Specializations:**
      - research_expert: Deep research and analysis
      - creative_writer: Content creation and writing
      - problem_solver: Technical and analytical problems

      **Always explain your approach** so users understand your process.
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
    sub_agents: [research_expert, creative_writer, problem_solver]
    think: true
    add_date: true

  research_expert:
    name: research_expert
    model: claude_research
    description: Expert researcher with web access and analytical thinking
    instruction: |
      You are a research expert specializing in comprehensive analysis.

      **Research Excellence:**
      - Use web search for the most current information
      - Cross-reference multiple sources for accuracy
      - Think through complex research questions systematically
      - Provide well-structured, evidence-based conclusions

      **Standards:**
      - Always cite your sources
      - Distinguish between facts and interpretations
      - Acknowledge limitations in available data
      - Present balanced perspectives when appropriate
    tools:
      - type: mcp
        command: npx
        args: ["-y", "@modelcontextprotocol/server-brave-search"]
    think: true

  creative_writer:
    name: creative_writer
    model: gpt4_creative
    description: Creative writing specialist with strategic thinking
    instruction: |
      You are a creative writing expert who thinks strategically about content.

      **Creative Process:**
      - Think through the writing goals and audience first
      - Consider tone, style, and structure options
      - Create engaging, well-structured content
      - Focus on clear communication and impact

      **Writing Expertise:**
      - Blog posts, articles, and essays
      - Marketing copy and social content
      - Technical documentation
      - Creative and narrative writing
    think: true

  problem_solver:
    name: problem_solver
    model: claude_analytical
    description: Analytical problem-solving specialist
    instruction: |
      You are an analytical problem solver who excels at complex challenges.

      **Problem-Solving Approach:**
      - Think through problems systematically
      - Break complex issues into manageable parts
      - Consider multiple solution approaches
      - Provide practical, actionable recommendations

      **Specializations:**
      - Technical and engineering problems
      - Business strategy and operations
      - Data analysis and interpretation
      - Process optimization and improvement
    think: true

models:
  claude_coordinator:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.5
    max_tokens: 3000

  claude_research:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.3
    max_tokens: 4000

  gpt4_creative:
    type: openai
    model: gpt-4o
    temperature: 0.8
    max_tokens: 3000

  claude_analytical:
    type: anthropic
    model: claude-3-5-sonnet-latest
    temperature: 0.2
    max_tokens: 3000
```

### Step 4: Test Your Ultimate System

```bash
# Run the ultimate system
./cagent -config ultimate-agent.yaml

# Try challenging requests:
# "Research and write a comprehensive analysis of remote work trends"
# "Help me solve a complex business optimization problem"
# "Create a strategic plan for launching a new product"
```

## 🎓 What You've Learned

Congratulations! You've built a complete multi-agent system from scratch. Let's review what you've accomplished:

### ✅ Key Skills Mastered

1. **Basic Agent Configuration** - Created simple, focused agents
2. **Tool Integration** - Added external capabilities with MCP tools
3. **Multi-Agent Coordination** - Built teams of specialized agents
4. **Advanced Features** - Used thinking capabilities and date context
5. **System Architecture** - Designed sophisticated agent hierarchies

### 🏗️ Progressive Architecture

You progressed through increasingly sophisticated architectures:

```
Basic Agent
    ↓
Agent + Tools
    ↓
Multi-Agent Team
    ↓
Advanced System with Thinking
```

Each level added new capabilities while maintaining the core principles.

### 💡 Key Lessons

**1. Instructions Are Everything**

- Clear, specific instructions produce better results
- Always document available tools and capabilities
- Explain when and how to delegate to sub-agents

**2. Specialization Beats Generalization**

- Focused agents outperform general-purpose ones
- Multiple simple agents can handle complex workflows
- Coordination agents route tasks to appropriate specialists

**3. Tools Extend Capabilities**

- Web search enables current information access
- File operations allow persistent workflows
- Custom tools can integrate any external system

**4. Thinking Improves Quality**

- Complex problems benefit from structured reasoning
- The think tool helps agents plan before responding
- Metacognitive approaches produce better results

## 🚀 Next Steps

### Immediate Actions

1. **Experiment** with the configurations you've created
2. **Modify** instructions to see how behavior changes
3. **Add** new tools from the MCP ecosystem
4. **Create** your own specialized agents

### Advanced Projects

1. **Build domain-specific teams** (e.g., financial analysis, content marketing)
2. **Integrate custom tools** for your specific workflows
3. **Design complex pipelines** with sequential processing
4. **Explore automated workflows** with file operations

### Learning Resources

- **[How-to Guide](./howto.md)** - More practical examples and patterns
- **[Explanation](./explanation.md)** - Deep dive into architecture and concepts
- **[Reference](./reference.md)** - Complete configuration documentation
- **[Examples](../examples/)** - Ready-to-use configurations

## 🎯 Best Practices Recap

1. **Start Simple**: Begin with basic agents, add complexity gradually
2. **Document Everything**: Clear instructions prevent confusion
3. **Test Iteratively**: Verify each component before adding more
4. **Think Like a Team**: Design agents as team members with clear roles
5. **Monitor Performance**: Watch how agents behave and refine accordingly

## 🔄 Common Patterns

You now understand these key patterns:

- **Router Pattern**: Central coordinator delegates to specialists
- **Pipeline Pattern**: Sequential processing through multiple agents
- **Tool Pattern**: Agents enhanced with external capabilities
- **Thinking Pattern**: Metacognitive reasoning for complex problems

## 🎉 You're Ready!

You now have the foundation to build sophisticated AI systems with cagent. The key is to start with your specific use case and apply the patterns you've learned.

**Remember**: Great agent systems are built iteratively. Start simple, test thoroughly, and add complexity only when needed.

**Happy building!** 🚀
