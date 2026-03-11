---
title: "Think Tool"
description: "Step-by-step reasoning scratchpad for planning and decision-making."
permalink: /tools/think/
---

# Think Tool

_Step-by-step reasoning scratchpad for planning and decision-making._

## Overview

The think tool is a reasoning scratchpad that lets agents think step-by-step before acting. The agent can write its thoughts without producing visible output to the user — ideal for planning complex tasks, breaking down problems, and reasoning through multi-step solutions.

This is a lightweight tool with no side effects. It's recommended for all agents — it improves the quality of reasoning on complex tasks at minimal cost.

## Configuration

```yaml
toolsets:
  - type: think
```

No configuration options.

<div class="callout callout-tip">
<div class="callout-title">💡 Tip
</div>
  <p>Include the think tool in every agent. It adds minimal overhead while significantly improving reasoning quality on complex tasks.</p>
</div>
