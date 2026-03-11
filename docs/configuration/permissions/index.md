---
title: "Permissions"
description: "Control which tools can execute automatically, require confirmation, or are blocked entirely."
permalink: /configuration/permissions/
---

# Permissions

_Control which tools can execute automatically, require confirmation, or are blocked entirely._

## Overview

Permissions provide fine-grained control over tool execution. You can configure which tools are auto-approved (run without asking), which require user confirmation, and which are completely blocked.

<div class="callout callout-info">
<div class="callout-title">ℹ️ Evaluation Order
</div>
  <p>Permissions are evaluated in this order: **Deny → Allow → Ask**. Deny patterns take priority, then allow patterns, and anything else defaults to asking for user confirmation.</p>

</div>

## Configuration

```yaml
agents:
  root:
    model: openai/gpt-4o
    description: Agent with permission controls
    instruction: You are a helpful assistant.

permissions:
  # Auto-approve these tools (no confirmation needed)
  allow:
    - "read_file"
    - "read_*" # Glob patterns
    - "shell:cmd=ls*" # With argument matching

  # Block these tools entirely
  deny:
    - "shell:cmd=sudo*"
    - "shell:cmd=rm*-rf*"
    - "dangerous_tool"
```

## Pattern Syntax

Permissions support glob-style patterns with optional argument matching:

### Simple Patterns

| Pattern        | Matches                        |
| -------------- | ------------------------------ |
| `shell`        | Exact match for `shell` tool   |
| `read_*`       | Any tool starting with `read_` |
| `mcp:github:*` | Any GitHub MCP tool            |
| `*`            | All tools                      |

### Argument Matching

You can match tools based on their argument values using `tool:arg=pattern` syntax:

```yaml
permissions:
  allow:
    # Allow shell only when cmd starts with "ls" or "cat"
    - "shell:cmd=ls*"
    - "shell:cmd=cat*"

    # Allow edit_file only in specific directory
    - "edit_file:path=/home/user/safe/*"

  deny:
    # Block shell with sudo
    - "shell:cmd=sudo*"

    # Block writes to system directories
    - "write_file:path=/etc/*"
    - "write_file:path=/usr/*"
```

### Multiple Argument Conditions

Chain multiple argument conditions with colons. All conditions must match:

```yaml
permissions:
  allow:
    # Allow shell with ls in current directory
    - "shell:cmd=ls*:cwd=."

  deny:
    # Block shell with rm -rf anywhere
    - "shell:cmd=rm*:cmd=*-rf*"
```

## Glob Pattern Rules

Patterns follow filepath.Match semantics with some extensions:

- `*` — matches any sequence of characters (including spaces)
- `?` — matches any single character
- `[abc]` — matches any character in the set
- `[a-z]` — matches any character in the range

Matching is **case-insensitive**.

<div class="callout callout-tip">
<div class="callout-title">💡 Trailing Wildcards
</div>
  <p>Trailing wildcards like <code>sudo*</code> match any characters including spaces, so <code>sudo*</code> matches <code>sudo rm -rf /</code>.</p>

</div>

## Decision Types

| Decision  | Behavior                                            |
| --------- | --------------------------------------------------- |
| **Allow** | Tool executes immediately without user confirmation |
| **Ask**   | User must confirm before tool executes (default)    |
| **Deny**  | Tool is blocked and returns an error to the agent   |

## Examples

### Read-Only Agent

Allow all read operations, block all writes:

```yaml
permissions:
  allow:
    - "read_file"
    - "read_multiple_files"
    - "list_directory"
    - "directory_tree"
    - "search_files_content"
  deny:
    - "write_file"
    - "edit_file"
    - "shell"
```

### Safe Shell Agent

Allow specific safe commands, block dangerous ones:

```yaml
permissions:
  allow:
    - "shell:cmd=ls*"
    - "shell:cmd=cat*"
    - "shell:cmd=grep*"
    - "shell:cmd=find*"
    - "shell:cmd=head*"
    - "shell:cmd=tail*"
    - "shell:cmd=wc*"
  deny:
    - "shell:cmd=sudo*"
    - "shell:cmd=rm*"
    - "shell:cmd=mv*"
    - "shell:cmd=chmod*"
    - "shell:cmd=chown*"
```

### MCP Tool Permissions

Control MCP tools by their qualified names:

```yaml
permissions:
  allow:
    # Allow all GitHub read operations
    - "mcp:github:get_*"
    - "mcp:github:list_*"
    - "mcp:github:search_*"
  deny:
    # Block destructive GitHub operations
    - "mcp:github:delete_*"
    - "mcp:github:close_*"
```

## Combining with Hooks

Permissions work alongside [hooks]({{ '/configuration/hooks/' | relative_url }}). The evaluation order is:

1. Check **deny** patterns — if matched, tool is blocked
2. Check **allow** patterns — if matched, tool is auto-approved
3. Run **pre_tool_use hooks** — hooks can allow, deny, or ask
4. If no decision, **ask user** for confirmation

Hooks can override allow decisions but cannot override deny decisions.

<div class="callout callout-warning">
<div class="callout-title">⚠️ Security Note
</div>
  <p>Permissions are enforced client-side. They help prevent accidental operations but should not be relied upon as a security boundary for untrusted agents. For stronger isolation, use <a href="{{ '/configuration/sandbox/' | relative_url }}">sandbox mode</a>.</p>

</div>
