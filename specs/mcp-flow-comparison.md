# MCP Flow Comparison

This document contrasts a standard MCP client-server interaction with the MCPHub flow,
and shows the extra twist MCPHub adds on top of MCP Shield: **per-MCP proxied URLs** and
**bundles** (curated, aggregated tool endpoints).

## Flow Diagram

```mermaid
graph TD
    subgraph "MCPHub Flow"
        Client2[Client e.g., Cursor / Pi]
        Hub[MCPHub Proxy]
        subgraph Endpoints
            McpURL[Per-MCP URL: /mcp/github]
            BundleURL[Bundle URL: /bundle/devtools]
        end
        Mcp1[Managed MCP: github]
        Mcp2[Managed MCP: jira]
        Mcp3[Managed MCP: weather]
        Service2[External Services]

        Client2 -- "OAuth2 / service token" --> Hub
        Hub --> McpURL
        Hub --> BundleURL
        McpURL --> Mcp1
        BundleURL --> Mcp1
        BundleURL --> Mcp2
        Mcp1 -- "rendered secret" --> Service2
        Mcp2 -- "rendered secret" --> Service2
        Service2 --> Mcp1
        Mcp1 --> Hub
        Hub --> Client2
    end

    subgraph "Standard MCP Flow"
        Client1[Client e.g., Cursor]
        MCPServer1[Directly Configured MCP Server]
        Service1[External Service e.g., GitHub API]

        Client1 -- "Invoke Tool" --> MCPServer1
        MCPServer1 -- "Authenticate & Call Service" --> Service1
        Service1 -- "Return Data" --> MCPServer1
        MCPServer1 -- "Return Result" --> Client1
    end
```

## Two ways to consume MCPHub

1. **Per-MCP URL.** Every `Mcp` resource is reachable at its own proxied URL
   (e.g. `/mcp/github`). The client sees that single server's tools (subject to the
   `Mcp`'s allow/deny and the caller's policy). This is the drop-in "managed single
   server" mode.

2. **Bundle URL.** An `McpBundle` aggregates selected tools from one or more `Mcp`s behind
   one endpoint (e.g. `/bundle/devtools`). The client sees a curated, renamed tool surface
   that may span GitHub + Jira + an internal MCP. See [bundles.md](bundles.md).

## Key Differences

| Feature | Standard Flow | MCPHub Flow |
|---------|---------------|-------------|
| **Configuration** | Client configures each MCP server individually, from local files. | MCPs are declared as `Mcp` CRs in cluster state; clients point at a proxied URL or bundle. |
| **Config source** | A local config file per client. | Kubernetes resources + controller state — no single YAML source. |
| **Tool discovery** | One configured server at a time. | Per-MCP, or aggregated/curated via bundles; intent-based search across the catalog. |
| **Endpoint** | One endpoint per server. | One endpoint per MCP **and** composable bundle endpoints. |
| **Authentication** | Per-server, often credentials in client config. | Centralized OAuth2 (users) + service tokens (machines). |
| **Authorization** | By the MCP server, if at all. | Centralized, fine-grained `McpPolicy` (RBAC) in cluster. |
| **Secrets** | Often exposed client-side. | Referenced by `Mcp`, rendered in-cluster only, never exposed to clients. |
| **Approvals** | None. | Optional human-in-the-loop pause/resume on sensitive tools. See [features.md](features.md). |
