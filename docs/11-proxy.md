# The MCP Proxy

The proxy is MCPHub's data plane. Every tool call from an agent flows through it. It exposes
two kinds of endpoint and does the same work on each request: authenticate, authorize, render
the upstream credential, route, and (if needed) pause for approval.

## Endpoints

- **Per-connector URL** — every [`Connector`](crds.md#mcpconnector) is reachable at
  `/mcp/<name>` (e.g. `/mcp/grafana-read`, `/mcp/grafana-write`). The client sees that
  connector's exposed, credentialed tool set. Several connectors can front the same provider
  with different credentials and scopes.
- **Bundle URL** — every [`Bundle`](crds.md#bundle) is reachable at `/bundle/<name>`.
  The client sees a curated, possibly renamed surface spanning multiple connectors. See
  [Bundles](bundles.md).

Agents point an MCP client **directly** at a `/mcp/...` or `/bundle/...` URL over Streamable
HTTP. For clients that can only launch a stdio command, the CLI's MCP server
(`mcphub mcp serve --transport stdio`) presents these endpoints locally and forwards to the
proxy.

## Per-request flow

```mermaid
sequenceDiagram
  participant A as Agent
  participant P as Proxy
  participant Pol as Policy (Cedar, cached)
  participant U as Upstream MCP
  A->>P: tool call (Bearer credential)
  P->>P: authenticate → subject
  P->>Pol: decision for subject + tool
  alt allow
    P->>U: forward (owner's secret rendered)
    U->>P: result
    P->>A: result
  else require approval
    P->>A: paused (execution id)
  else block
    P->>A: 403
  end
```

1. **Authenticate** the caller (OIDC token, tokenless-minted credential, or service token)
   to a subject. See [Authentication](authentication.md).
2. **Authorize** against the in-memory Cedar policy. See [Authorization](authorization.md).
3. **Render the upstream credential** in-cluster and inject it into the upstream MCP. The
   caller never sees it.
4. **Route** the (possibly renamed) tool to the correct upstream MCP and forward.
5. **Pause** if the tool requires approval, returning an execution id; resume on approval.

## Tool naming

- Per-MCP: tools appear with the MCP's `exposePrefix` (e.g. `github_`).
- Bundle: tools appear with the bundle's `as` rename template; the proxy maps each back to
  `(source MCP, upstream tool)`.

## Discovery

The proxy also exposes a discovery surface so agents can find tools by intent at runtime
(backed by the unified catalog). The same search is available in the [Web UI](web-ui.md) and
[CLI](cli.md).

## Comparison with a direct MCP connection

| | Direct MCP | Through MCPHub |
|--|-----------|----------------|
| Config | Per server, in client | One URL (per-MCP or bundle) |
| Auth | Per server, often secrets in client config | Central OIDC / tokenless / token |
| Authorization | By the server, if any | Central policy, per tool |
| Secrets | Often client-side | Owned by the connector, rendered in-cluster |
| Approvals | None | Human-in-the-loop on sensitive tools |
| Aggregation | One server | Per-MCP **and** composable bundles |
