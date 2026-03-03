---
title: "Agent Distribution"
description: "Package, share, and run agents via OCI-compatible registries — just like container images."
permalink: /concepts/distribution/
---

# Agent Distribution

_Package, share, and run agents via OCI-compatible registries — just like container images._

## Overview

cagent agents can be pushed to any OCI-compatible registry (Docker Hub, GitHub Container Registry, etc.) and pulled/run anywhere. This makes sharing agents as easy as sharing Docker images.

<div class="callout callout-tip">
<div class="callout-title">💡 Tip
</div>
  <p>For CLI commands related to distribution, see <a href="/features/cli/">CLI Reference</a> (<code>cagent share push</code>, <code>cagent share pull</code>, <code>cagent alias</code>).</p>

</div>

## Pushing Agents

```bash
# Push to Docker Hub
$ cagent share push ./agent.yaml docker.io/username/my-agent:latest

# Push to GitHub Container Registry
$ cagent share push ./agent.yaml ghcr.io/username/my-agent:v1.0
```

## Pulling Agents

```bash
# Pull an agent
$ cagent pull docker.io/username/my-agent:latest

# Pull from the agent catalog
$ cagent pull agentcatalog/pirate
```

## Running from a Registry

Run agents directly from a registry without pulling first:

```bash
# Run directly from Docker Hub
$ cagent run docker.io/username/my-agent:latest

# Run from the agent catalog
$ cagent run agentcatalog/pirate

# Run with a specific agent from a multi-agent config
$ cagent run docker.io/username/dev-team:latest -a developer
```

## Agent Catalog

The `agentcatalog` namespace on Docker Hub hosts pre-built agents you can try:

```bash
# Try the pirate-themed assistant
$ cagent run agentcatalog/pirate

# Try the coding agent
$ cagent run agentcatalog/coder
```

## Using with Aliases

Combine OCI references with aliases for convenient access:

```bash
# Create an alias for a registry agent
$ cagent alias add coder agentcatalog/coder --yolo

# Now just run
$ cagent run coder
```

## Using with API Server

The API server supports OCI references with auto-refresh:

```bash
# Start API from registry, auto-pull every 10 minutes
$ cagent api docker.io/username/agent:latest --pull-interval 10
```

## Private Repositories

cagent supports pulling from private GitHub repositories and registries that require authentication. Use standard Docker login or GitHub authentication:

```bash
# Login to a registry
$ docker login docker.io

# Now push/pull works with private repos
$ cagent share push ./agent.yaml docker.io/myorg/private-agent:latest
$ cagent run docker.io/myorg/private-agent:latest
```

<div class="callout callout-info">
<div class="callout-title">ℹ️ Troubleshooting
</div>
  <p>Having issues with push/pull? See <a href="/community/troubleshooting/">Troubleshooting</a> for common registry issues.</p>

</div>
