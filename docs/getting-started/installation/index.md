---
title: "Installation"
description: "Get cagent running on your system in minutes."
permalink: /getting-started/installation/
---

# Installation

_Get cagent running on your system in minutes._

## Prerequisites

- An API key for at least one AI provider (OpenAI, Anthropic, Google, etc.)
- **Optional:** [Docker Desktop](https://www.docker.com/products/docker-desktop/) — for running containerized MCP tools and Docker Model Runner

## Docker Desktop (Pre-installed)

Starting with [Docker Desktop 4.49.0](https://docs.docker.com/desktop/release-notes/#4490), **cagent is already available**. No separate installation needed — just open a terminal and run:

```bash
cagent version
```

<div class="callout callout-tip">
<div class="callout-title">💡 Tip
</div>
  <p>Docker Desktop bundles cagent and keeps it up to date. This is the easiest way to get started, especially if you want to use Docker MCP tools and Docker Model Runner.</p>

</div>

## Homebrew (macOS / Linux)

Install cagent using [Homebrew](https://brew.sh/):

```bash
# Install
$ brew install docker/tap/cagent

# Verify
$ cagent version
```

## Download Binary Releases

Download [prebuilt binary releases](https://github.com/docker/cagent/releases) for Windows, macOS, and Linux from the GitHub Releases page.

### macOS / Linux

```bash
# Download the latest release (adjust URL for your platform)
curl -L https://github.com/docker/cagent/releases/latest/download/cagent-$(uname -s)-$(uname -m) -o cagent
chmod +x cagent
sudo mv cagent /usr/local/bin/
```

### Windows

Download `cagent-Windows-x86_64.exe` from the [releases page](https://github.com/docker/cagent/releases) and add it to your PATH.

## Build from Source

For the latest features, or to contribute, build from source:

### Prerequisites

- [Go 1.25](https://go.dev/dl/) or higher
- [Task 3.44](https://taskfile.dev/installation/) or higher (build tool)
- [golangci-lint](https://golangci-lint.run/docs/welcome/install/#binaries) (for linting)

```bash
# Clone the repository
git clone https://github.com/docker/cagent.git
cd cagent

# Build the binary
task build

# The binary is at ./bin/cagent
./bin/cagent --help
```

<div class="callout callout-tip">
<div class="callout-title">💡 Building on Windows
</div>
  <p>On Windows, use <code>task build-local</code> instead of <code>task build</code>. This builds the binary inside a Docker container using Docker Buildx, which avoids issues with Windows-specific toolchain setup and CGo cross-compilation. The output goes to the <code>./dist</code> directory.</p>

</div>

## Set Up API Keys

cagent needs API keys for the model providers you want to use. Set them as environment variables:

```bash
# Pick one (or more) depending on your provider
export OPENAI_API_KEY="sk-..."           # OpenAI
export ANTHROPIC_API_KEY="sk-ant-..."    # Anthropic
export GOOGLE_API_KEY="AI..."           # Google Gemini
export MISTRAL_API_KEY="..."            # Mistral
```

<div class="callout callout-info">
<div class="callout-title">ℹ️ Note
</div>
  <p>You only need the key(s) for the provider(s) you configure in your agent YAML. If you use Docker Model Runner (DMR), no API key is needed — models run locally.</p>

</div>

## Verify Installation

```bash
# Check the version
$ cagent version

# Run the default agent
$ cagent run

# Or try a built-in example
$ cagent run agentcatalog/pirate
```

## What's Next?

<div class="cards">
  <a class="card" href="/getting-started/quickstart/">
    <div class="card-icon">⚡</div>
    <h3>Quick Start</h3>
    <p>Create and run your first agent in under 5 minutes.</p>
  </a>
  <a class="card" href="/community/troubleshooting/">
    <div class="card-icon">🔧</div>
    <h3>Troubleshooting</h3>
    <p>Something not working? Debug mode, common issues, and solutions.</p>
  </a>
</div>
