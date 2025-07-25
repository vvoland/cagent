# LiteLLM example

This demonstrates how cagent could talk to an AI Gateway
that proxies both LLM calls and MCP tool calls.

The main advantage is that all the credentials can then
be handled centrally, on the Gateway side.

## Run the LiteLLM based AI Gateway

Make sure `OPENAI_API_KEY` and `ANTHROPIC_API_KEY` env variables are configured
in `./ai-gateway/.env`.

Make sure `brave.api_key` env variable is configured in `./ai-gateway/.env.mcp`.

```
docker compose --project-directory ai-gateway up -d
```

## Configure an LLM Virtual Key on LiteLLM's console

+ Open http://localhost:4000/ui
+ Login with user `admin` and password `sk-1234`
+ Open http://localhost:4000/ui/?page=api-keys
+ Click `Create New Key`
+ Pick a name. e.g `My Key`
+ Models is `All Team Models`
+ Click on `Create Key`
+ Copy the value

## Run the agent without the AI Gateway

```
export OPENAI_API_KEY="<YOUR_OPENAI_API_KEY>"
cagent run pirate.yaml

export ANTHROPIC_API_KEY="<YOUR_ANTHROPIC_API_KEY>"
cagent run web_searcher.yaml
```

## Run the agent with the AI Gateway

```
export LITELLM_API_KEY="<YOUR_LITELLM_API_KEY>"
cagent run --gateway=http://localhost:4000 pirate.yaml
cagent run --gateway=http://localhost:4000 web_searcher.yaml
```