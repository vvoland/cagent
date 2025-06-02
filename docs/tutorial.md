# Agent Tutorial: Creating Your First Intelligent Agent

Welcome to this step-by-step tutorial that will guide you through creating your
first intelligent agent. By the end of this tutorial, you'll have a working
agent that can respond to queries with a distinct personality and access
external tools.

## Prerequisites

Before starting, ensure you have:

- Basic understanding of YAML syntax
- A text editor for creating and editing YAML files
- Access to language model APIs (if running your own deployment)

## Step 1: Set Up Your Project Directory

First, let's create a directory for our project:

```bash
mkdir my-first-agent
cd my-first-agent
```

## Step 2: Create a Basic Agent Configuration

Create a file named `agent.yaml` in your project directory and add the following
basic configuration:

```yaml
agents:
  root:
    name: assistant
    model: openai
    description: A helpful assistant
    instruction: |
      You are a helpful assistant that provides clear and concise information.
      Always be polite and professional in your responses.

models:
  openai:
    type: openai
    model: gpt-4o
    temperature: 0.7
```

This defines a simple agent that will respond as a helpful assistant using
OpenAI's GPT-4o model.

## Step 3: Enhance the Agent with Personality

Let's modify our agent to have a more distinct personality. Update the
`instruction` field:

```yaml
instruction: |
  You are a cheerful and enthusiastic assistant who loves to help people.
  Always respond with energy and positivity, using exclamation points and upbeat language.
  Break down complex topics into simple, easy-to-understand explanations.

  **Constraints:**
  * Use markdown formatting to highlight important information
  * Keep responses concise but thorough
  * Always offer a positive encouragement at the end of your responses
```

## Step 4: Add a Simple Tool

Now, let's add a tool that allows our agent to search for information. Update
your `agent.yaml` file:

```yaml
agents:
  root:
    name: assistant
    model: openai
    description: A cheerful and helpful assistant
    instruction: |
      You are a cheerful and enthusiastic assistant who loves to help people.
      Always respond with energy and positivity, using exclamation points and upbeat language.
      Break down complex topics into simple, easy-to-understand explanations.

      <TASK>
        # **Workflow:**
        # 1. Understand the user's question
        # 2. If information is needed, use the search tool
        # 3. Provide a clear, enthusiastic response
      </TASK>

      **Tools:**
      You have access to the following tools:
      * `search(query: str) -> str`: Searches the web for information

      **Constraints:**
      * Use markdown formatting to highlight important information
      * Keep responses concise but thorough
      * Always offer a positive encouragement at the end of your responses
    tools:
      - type: search

models:
  openai:
    type: openai
    model: gpt-4o
    temperature: 0.7

tools:
  search:
    type: mcp
    command: npx
    args: ["-y", "search-tool"]
```

## Step 5: Create a Sub-Agent

Let's add a fact-checking sub-agent that our main agent can consult:

```yaml
agents:
  root:
    name: assistant
    model: openai
    description: A cheerful and helpful assistant
    instruction: |
      You are a cheerful and enthusiastic assistant who loves to help people.
      Always respond with energy and positivity, using exclamation points and upbeat language.
      Break down complex topics into simple, easy-to-understand explanations.

      <TASK>
        # **Workflow:**
        # 1. Understand the user's question
        # 2. If information is needed, use the search tool
        # 3. For factual claims, consult the fact_checker sub-agent
        # 4. Provide a clear, enthusiastic response
      </TASK>

      **Tools:**
      You have access to the following tools:
      * `search(query: str) -> str`: Searches the web for information

      **Constraints:**
      * Use markdown formatting to highlight important information
      * Keep responses concise but thorough
      * Always offer a positive encouragement at the end of your responses
    tools:
      - type: search
    sub_agents:
      - fact_checker

  fact_checker:
    name: fact_checker
    model: openai
    description: Verifies factual accuracy of information
    instruction: |
      You are a careful and methodical fact checker.
      Your job is to verify claims by examining evidence and providing a confidence assessment.

      For each claim you review:
      1. Examine the available evidence
      2. Research additional information if needed
      3. Provide a confidence rating from 1-5
      4. Explain your reasoning briefly

      **Tools:**
      You have access to the following tools:
      * `search(query: str) -> str`: Searches the web for information

models:
  openai:
    type: openai
    model: gpt-4o
    temperature: 0.7

tools:
  search:
    type: mcp
    command: npx
    args: ["-y", "search-tool"]
```

## Step 6: Enable the Think Tool

Finally, let's enable the "think" tool to help our agent reason through complex
problems:

```yaml
agents:
  root:
    name: assistant
    model: openai
    description: A cheerful and helpful assistant
    think: true
    instruction: |
      You are a cheerful and enthusiastic assistant who loves to help people.
      Always respond with energy and positivity, using exclamation points and upbeat language.
      Break down complex topics into simple, easy-to-understand explanations.

      When faced with complex problems, use the think tool to reason through your approach before responding.

      <TASK>
        # **Workflow:**
        # 1. Understand the user's question
        # 2. For complex questions, use the think tool to plan your approach
        # 3. If information is needed, use the search tool
        # 4. For factual claims, consult the fact_checker sub-agent
        # 5. Provide a clear, enthusiastic response
      </TASK>

      **Tools:**
      You have access to the following tools:
      * `search(query: str) -> str`: Searches the web for information
      * `think(thought: str) -> None`: Helps you reason through complex problems

      **Constraints:**
      * Use markdown formatting to highlight important information
      * Keep responses concise but thorough
      * Always offer a positive encouragement at the end of your responses
    tools:
      - type: search
    sub_agents:
      - fact_checker

  # fact_checker configuration remains the same
```

## Step 7: Test Your Agent

Once your agent configuration is complete, you can test it by running:

```bash
# Command will vary based on your specific deployment method
agent run agent.yaml
```

Try asking your agent different types of questions to see how it responds:

1. Simple questions: "What's the weather like today?"
2. Questions requiring research: "What are the benefits of meditation?"
3. Complex problems: "How would you design a system to reduce traffic congestion
   in a city?"

## Next Steps

Congratulations! You've created your first intelligent agent with a distinct
personality, tools, and sub-agents. Here are some ways to further enhance your
agent:

1. Add more specialized tools for specific tasks
2. Create additional sub-agents for different domains
3. Refine the instructions to improve the agent's responses
4. Implement more complex workflows for specific use cases

For more advanced configurations and options, refer to the [How-to
Guide](./howto.md) and [Reference Documentation](./reference.md).

For conceptual understanding of how agents work, see the
[Explanation](./explanation.md) section.
