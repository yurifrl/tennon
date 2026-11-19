# Tennon

The Kubernetes-native control plane and proxy for Model Context Protocol (MCP) servers.

Register MCP servers, share them safely (credentials never leave the platform), and govern
every tool call — driven from a web UI, CLI, or API that all write the same objects.

**📖 Documentation: <https://yurifrl.github.io/tennon/>**

| | |
|---|---|
| [Getting Started](https://yurifrl.github.io/tennon/#/?id=getting-started) | Install, first connector, connect an agent |
| [Concepts](https://yurifrl.github.io/tennon/#/?id=concepts) | Connector, tool, tag, subject, policy, token |
| [CLI](https://yurifrl.github.io/tennon/#/?id=cli) · [API](https://yurifrl.github.io/tennon/#/?id=api) | `tennon`, `tctl`, and the HTTP surface |
| [Architecture](https://yurifrl.github.io/tennon/#/arch/) | Design, internals, and the [roadmap](https://yurifrl.github.io/tennon/#/arch/?id=roadmap) |

The same pages are browsable in the repo under [`docs/`](docs/00-README.md) and
[`docs/arch/`](docs/arch/README.md).

## How it works

```
 MCP client ──▶  tennon proxy  ──▶  upstream MCP server
 (no secret)     /mcp/<name>        (credential injected here)
                      ▲
        tctl / API / UI register connectors and secrets
```

You register an MCP server once as a **connector**, with its credential stored as a
**secret**. Agents call the connector through Tennon's proxy; the proxy injects the credential
server-side, so callers never hold it.

## Status

Phase 1 of the [roadmap](docs/arch/10-roadmap.md) is implemented: **register a connector,
reconcile it, and proxy its tools at `/mcp/<name>` using the connector's rendered credential —
never exposing that credential to the caller.** Later phases (spec connectors, pluggable
storage/auth, Cedar authorization, tokenless identity, approvals, and the React UI) are in
progress, so some pages in the docs describe behavior that has not shipped yet.

## Install

Tennon is two binaries: the server (`tennon`) and the client (`tctl`). With Go 1.26+:

```bash
go install github.com/yurifrl/tennon/cmd/tennon@latest
go install github.com/yurifrl/tennon/cmd/tctl@latest
```

## Quickstart

Run the control plane locally with an in-memory store. With no OIDC issuer configured, every
caller is anonymous and every call is allowed, which is what you want for a first try:

```bash
tennon --store memory --runtime remote \
  --api-addr 127.0.0.1:8080 --proxy-addr 127.0.0.1:8081
```

In a second terminal, start a mock MCP upstream to proxy to (it exposes two tools,
`get_status` and `delete_all`):

```bash
go run github.com/yurifrl/tennon/hack/mockmcp@latest --addr 127.0.0.1:9090
```

In a third, store a credential and register the upstream as a connector:

```bash
export TENNON_API=http://127.0.0.1:8080

tctl secret create grafana-creds --from-literal token=$GRAFANA_TOKEN
# secret "grafana-creds" created (1 keys)

tctl connector add grafana \
  --remote http://127.0.0.1:9090/mcp \
  --header "Authorization=Bearer {{secret:grafana-creds.token}}" \
  --expose-prefix grafana_ --allow "get_*" --tag observability
# connector "grafana" created (remote)

tctl connector list
# grafana                  remote     Ready            /mcp/grafana
```

The connector is now served at `http://127.0.0.1:8081/mcp/grafana`.

## Examples

### Connect an agent

Point any MCP client that speaks Streamable HTTP at the connector's proxy URL. No token or
upstream credential goes in the client config:

```json
{
  "mcpServers": {
    "grafana": {
      "type": "http",
      "url": "http://127.0.0.1:8081/mcp/grafana"
    }
  }
}
```

For a stdio-only client, bridge with `mcp-remote`:

```json
{
  "mcpServers": {
    "grafana": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "http://127.0.0.1:8081/mcp/grafana"]
    }
  }
}
```

### Check the proxy with curl

```bash
curl -si http://127.0.0.1:8081/mcp/grafana \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"curl","version":"0"}}}'
# HTTP/1.1 200 OK
# ...
# data: {"jsonrpc":"2.0","id":1,"result":{...,"serverInfo":{"name":"tennon/grafana","version":"0.1.0"}}}
```

### Limit and rename the tools a connector exposes

`--allow` and `--deny` take globs over the upstream's tool names; `--expose-prefix` namespaces
what the agent sees. The quickstart connector allows `get_*`, so the agent gets
`grafana_get_status` and never sees `delete_all`:

```bash
tctl connector describe grafana
# ...
#   "tools": { "exposePrefix": "grafana_", "allow": ["get_*"] },
#   "status": { "phase": "Ready", "discoveredTools": 1, "url": "/mcp/grafana" }
```

To expose everything except the destructive tools instead:

```bash
tctl connector add grafana-full \
  --remote http://127.0.0.1:9090/mcp \
  --header "Authorization=Bearer {{secret:grafana-creds.token}}" \
  --deny "delete_*"
```

### Keep state across restarts

The in-memory store is lost when the server stops. Use SQLite for a single host:

```bash
tennon --store sqlite --store-path tennon.db \
  --api-addr 127.0.0.1:8080 --proxy-addr 127.0.0.1:8081
```

`--store` also accepts `postgres` and `kubernetes`; see
[Storage backends](https://yurifrl.github.io/tennon/#/?id=storage-backends).

### Let an agent manage Tennon

`tctl` embeds an MCP server over the Tennon API, so an agent can manage connectors itself:

```json
{
  "mcpServers": {
    "tennon-admin": {
      "command": "tctl",
      "args": ["mcp", "serve"],
      "env": { "TENNON_API": "http://127.0.0.1:8080" }
    }
  }
}
```

### Clean up

```bash
tctl connector delete grafana
```

## Contributing

See [Contributing](docs/17-contributing.md) for the repository layout, build and test
commands, and conventions.
