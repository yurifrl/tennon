# CLI

`tctl` is a standalone Go binary (separate from the `tennon` server): a thin client over
the server API. It authenticates with an OIDC token or a service token and never touches the
store directly, so the same validation, policy, and audit apply as the UI.

The CLI also **embeds an MCP server** that exposes CLI operations as tools, allowing AI
agents to interact with Tennon programmatically through the Model Context Protocol.

Command names below are indicative of the planned surface; run `tctl --help` for the
authoritative list in your build.

## Auth

```bash
tctl login                       # OIDC device/browser flow → cached token
tctl login --token mh_svc_…      # use a service token (CI)
tennon whoami                      # show resolved subject + groups
```

## Connectors

```bash
tctl connector list
# an MCP server (TennonConnector)
tctl connector add github --container ghcr.io/github/github-mcp-server:latest \
  --transport stdio --expose-prefix github_ \
  --allow "search_*,get_*,list_*" --deny "delete_*" --tag code,prod
# an OpenAPI service (OpenApiConnector)
tctl connector add billing --openapi https://billing.acme.internal/openapi.json
tctl connector describe github
tctl connector refresh github          # re-discover/recompile tools
tctl connector delete github
```

## Secrets

```bash
tctl secret create github-credentials --from-literal=token=$GITHUB_TOKEN
# referenced by a connector as {{secret:github-credentials.token}}; value never printed back
```

## Sharing & policy

```bash
# share a connector (sugar that authors a Policy rule)
tctl connector share grafana \
  --to "group:okta:observability" \
  --tools "grafana_get_*,grafana_search_*"

# org-wide guardrail (Policy)
tctl policy list
tctl policy create sensitive-needs-approval \
  --where "tag:sensitive" --effect ask
```

## Tokens & service accounts

```bash
# tokenless (preferred)
tennon sa create hermes --owner platform-team \
  --trust oidc --issuer https://token.actions.githubusercontent.com \
  --audience tennon --subject "repo:acme/hermes:ref:refs/heads/main"

# fine-grained token (fallback)
tctl token create support-bot --grant "tag:chat" --ttl 1h
tctl token revoke support-bot
```

## Tools: discover, inspect, call, resume

```bash
tennon tools search "open a pull request"
tennon tools describe github_create_pull_request
tennon call github_get_issue '{"owner":"acme","repo":"app","number":12}'
tctl resume --execution-id exec_123    # approve/continue a paused call
```

## Embedded MCP Server

The CLI embeds an MCP server that exposes CLI operations as tools, enabling AI agents to
interact with Tennon programmatically:

```bash
# Start the embedded MCP server (stdio — for clients that launch a command)
tctl mcp serve
tctl mcp serve --transport stdio

# Or expose it over Streamable HTTP (SSE is not used anywhere in Tennon)
tctl mcp serve --transport streamable-http --port 8080

# Available tools include:
# - tennon_connector_list:   List connectors
# - tennon_connector_create: Create a connector
# - tennon_secret_create:    Create secrets
# - tennon_tool_call:        Call tools through Tennon
# - tennon_policy_manage:    Manage access policies
```

This allows AI coding assistants (like Claude Code, Cursor, etc.) to connect to Tennon
and perform operations through the MCP protocol without manual CLI usage.

## Output

Most commands accept `--json` for machine-readable output, suitable for scripting and CI.
