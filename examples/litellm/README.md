# LiteLLM example

This demonstrates how cagent could talk to an AI Gateway
that proxies both LLM calls and MCP tool calls.

The main advantage is that all the credentials can then
be handled centrally, on the Gateway side.

## Run the LiteLLM based AI Gateway

Make sure `MY_OPENAI_API_KEY` env variable is configured.

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

## Configure an MCP Server on LiteLLM's console

+ Open http://localhost:4000/ui
+ Login with user `admin` and password `sk-1234`
+ Open http://localhost:4000/ui/?page=mcp-servers
+ Click `Add New MCP Server`
+ Pick a name. e.g `Docker MCP Gateway`
+ Transport Type is `HTTP`
+ MCP Server URL is `http://gateway.com:9011`
+ Authentication is `None`
+ MCP Version is `... (Latest)`
+ Click on `Save Changes`

## Run then agent

```
export LITELLM_API_KEY="The key you copied"
cagent run pirate.yaml
cagent run web_searcher.yaml
```