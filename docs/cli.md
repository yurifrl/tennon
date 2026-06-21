# CLI

`mcphub` is a standalone Go binary: a thin client over the controller API. It authenticates
with an OIDC token or a service token and never touches the store directly, so the same
validation, policy, and audit apply as the UI.

The CLI also **embeds an MCP server** that exposes CLI operations as tools, allowing AI
agents to interact with MCPHub programmatically through the Model Context Protocol.

> Command names below are indicative of the planned surface; run `mcphub --help` for the
> authoritative list in your build.

## Auth

```bash
mcphub login                       # OIDC device/browser flow → cached token
mcphub login --token mh_svc_…      # use a service token (CI)
mcphub whoami                      # show resolved subject + groups
```

## MCPs

```bash
mcphub mcp list
mcphub mcp add github --container ghcr.io/github/github-mcp-server:latest \
  --transport stdio --expose-prefix github_ \
  --allow "search_*,get_*,list_*" --deny "delete_*" --tag code,prod
mcphub mcp describe github
mcphub mcp refresh github          # re-discover tools
mcphub mcp delete github
```

## Secrets

```bash
mcphub secret create github-credentials --from-literal=token=$GITHUB_TOKEN
# referenced by an MCP as {{secret:github-credentials.token}}; value never printed back
```

## Bundles

```bash
mcphub bundle list
mcphub bundle create devtools \
  --from github --select "search_*,get_*,list_*" --as "gh_{tool}" \
  --from jira --select "create_issue,get_issue,search" --as "jira_{tool}" \
  --require-approval "jira_create_issue"
```

## Policies

```bash
mcphub policy list
mcphub policy create share-grafana \
  --subject "group:okta:observability" \
  --grant "mcp:grafana:grafana_get_*,grafana_search_*"
```

## Tokens & service accounts

```bash
# tokenless (preferred)
mcphub sa create hermes --owner platform-team \
  --trust oidc --issuer https://token.actions.githubusercontent.com \
  --audience mcphub --subject "repo:acme/hermes:ref:refs/heads/main"

# fine-grained token (fallback)
mcphub token create support-bot --grant "tag:chat" --ttl 1h
mcphub token revoke support-bot
```

## Tools: discover, inspect, call, resume

```bash
mcphub tools search "open a pull request"
mcphub tools describe github_create_pull_request
mcphub call github_get_issue '{"owner":"acme","repo":"app","number":12}'
mcphub resume --execution-id exec_123    # approve/continue a paused call
```

## Embedded MCP Server

The CLI embeds an MCP server that exposes CLI operations as tools, enabling AI agents to
interact with MCPHub programmatically:

```bash
# Start the embedded MCP server (stdio — for clients that launch a command)
mcphub mcp serve
mcphub mcp serve --transport stdio

# Or expose it over Streamable HTTP (SSE is not used anywhere in MCPHub)
mcphub mcp serve --transport streamable-http --port 8080

# Available tools include:
# - mcphub_mcp_list: List available MCPs and connections
# - mcphub_mcp_create: Create a new MCP connection
# - mcphub_secret_create: Create secrets
# - mcphub_bundle_create: Create tool bundles
# - mcphub_tool_call: Call tools through MCPHub
# - mcphub_policy_manage: Manage access policies
```

This allows AI coding assistants (like Claude Code, Cursor, etc.) to connect to MCPHub
and perform operations through the MCP protocol without manual CLI usage.

## Output

Most commands accept `--json` for machine-readable output, suitable for scripting and CI.
