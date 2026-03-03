---
title: "Skills"
description: "Skills provide specialized instructions that agents can load on demand when a task matches a skill's description."
permalink: /features/skills/
---

# Skills

_Skills provide specialized instructions that agents can load on demand when a task matches a skill's description._

## How Skills Work

1. cagent scans standard directories for `SKILL.md` files
2. Skill metadata (name, description) is injected into the agent's system prompt
3. When a user request matches a skill, the agent reads the full instructions
4. The agent follows the skill's detailed instructions to complete the task

## Enabling Skills

```yaml
agents:
  root:
    model: openai/gpt-4o
    instruction: You are a helpful assistant.
    skills: true
    toolsets:
      - type: filesystem # required for reading skill files
```

<div class="callout callout-tip">
<div class="callout-title">💡 Tip
</div>
  <p>Skills are perfect for encoding team-specific workflows (PR review, deployment, coding standards) that apply across projects.</p>

</div>

## SKILL.md Format

```yaml
---
name: create-dockerfile
description: Create optimized Dockerfiles for applications
license: Apache-2.0
metadata:
  author: my-org
  version: "1.0"
---

# Creating Dockerfiles

When asked to create a Dockerfile:

1. Analyze the application type and language
2. Use multi-stage builds for compiled languages
3. Minimize image size by using slim base images
4. Follow security best practices (non-root user, etc.)
```

## Search Paths

Skills are discovered from these locations (later overrides earlier):

### Global

| Path                | Search Type                             |
| ------------------- | --------------------------------------- |
| `~/.codex/skills/`  | Recursive (searches all subdirectories) |
| `~/.claude/skills/` | Flat (immediate children only)          |
| `~/.agents/skills/` | Recursive (searches all subdirectories) |

### Project (from git root to current directory)

| Path              | Search Type                                |
| ----------------- | ------------------------------------------ |
| `.claude/skills/` | Flat (cwd only)                            |
| `.agents/skills/` | Flat (each directory from git root to cwd) |

## Invoking Skills

Skills can be invoked in multiple ways:

- **Automatic:** The agent detects when your request matches a skill's description and loads it automatically
- **Explicit:** Reference the skill name in your prompt: "Use the create-dockerfile skill to..."
- **Slash command:** Use `/{skill-name}` to invoke a skill directly

```bash
# In the TUI, invoke skill directly:
/create-dockerfile

# Or mention it in your message:
"Create a dockerfile for my Python app (use the create-dockerfile skill)"
```

## Precedence

When multiple skills share the same name:

1. Global skills load first
2. Project skills load next, from git root toward current directory
3. Skills closer to the current directory override those further away

## Creating a Skill

```bash
# Create the skill directory
$ mkdir -p ~/.agents/skills/create-dockerfile

# Write the SKILL.md file
$ cat > ~/.agents/skills/create-dockerfile/SKILL.md << 'EOF'
---
name: create-dockerfile
description: Create optimized Dockerfiles for applications
---

# Creating Dockerfiles

When asked to create a Dockerfile:

1. Analyze the application type and language
2. Use multi-stage builds for compiled languages
3. Use slim base images to minimize size
4. Run as non-root user for security
EOF
```

The skill will automatically be available to any agent with `skills: true`.

<div class="callout callout-info">
<div class="callout-title">ℹ️ See also
</div>
  <p>Skills are enabled in the <a href="/configuration/agents/">Agent Config</a> with the <code>skills: true</code> property. For tool-based capabilities, see <a href="/concepts/tools/">Tools</a>.</p>

</div>
