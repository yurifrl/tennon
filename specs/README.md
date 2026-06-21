# MCPHub

MCPHub is a Kubernetes-native control plane and proxy for Model Context Protocol (MCP)
servers. It is a Kubernetes controller (built with Kubebuilder) that also ships an API
and a web UI, turning a cluster into a managed catalog of MCP servers that agents can
discover, authorize against, and call through a single proxy endpoint.

It is heavily inspired by [MCP Shield](https://github.com/nsxbet/mcp-shield) (the proxy +
controller foundation) and [RhysSullivan/executor](https://github.com/RhysSullivan/executor)
(the unified tool catalog, intent discovery, and bundle/approval workflow), with several
deliberate twists.

## What makes MCPHub different

- **State-backed configuration, not a single YAML file.** MCP definitions are not read
  from one config file at boot. Each MCP is a Kubernetes `Mcp` custom resource; the
  controller reconciles those into runtime state, and the proxy reads from that state.
  See [manifest.md](manifest.md).
- **`Mcp` kind that renders secrets.** The `Mcp` CRD references Kubernetes `Secret`s and
  the controller renders them into the spawned server's environment / headers. Secret
  values never live in the catalog or leave the cluster boundary. See [manifest.md](manifest.md).
- **OAuth2 + service tokens.** End users authenticate via OAuth2 (authorization code +
  PKCE). Machine clients use long-lived service tokens. Both feed the same authorization
  layer. See [authentication-flow.md](authentication-flow.md).
- **MCP proxy with bundles.** Every MCP gets its own proxied URL. On top of that, you can
  compose **bundles** — curated sets of tools drawn from one or many MCPs, exposed as a
  single endpoint with its own auth and tool surface. See
  [mcp-flow-comparison.md](mcp-flow-comparison.md) and [bundles.md](bundles.md).
- **Executor-style catalog features.** Source auto-detection, intent-based tool discovery,
  pause/resume (human-in-the-loop) executions, and policy enforcement. See
  [features.md](features.md).

## Components

MCPHub ships as **two binaries**:

1. **`mcphub-controller`** — the controller and the API, together in one process. It
   reconciles `Mcp` / `McpBundle` / `McpPolicy` resources, serves the management + proxy
   API, and runs the MCP proxy.
2. **`mcphub-web`** — a thin server that serves the Vite-built web interface and proxies
   browser calls to the controller's API.

See [architecture.md](architecture.md) for the full breakdown.

## Documentation map

| Doc | Purpose |
|-----|---------|
| [architecture.md](architecture.md) | Binaries, processes, and how they fit together |
| [manifest.md](manifest.md) | The `Mcp` CRD, state-backed config, secret rendering |
| [authentication-flow.md](authentication-flow.md) | OAuth2, service tokens, authorization |
| [mcp-flow-comparison.md](mcp-flow-comparison.md) | Standard MCP vs. MCPHub proxy + bundles |
| [bundles.md](bundles.md) | Composing curated tool bundles |
| [features.md](features.md) | Executor-inspired catalog, discovery, approvals |
| [how-to-run.md](how-to-run.md) | Running the binaries locally and in a cluster |
| [roadmap.md](roadmap.md) | Phased delivery plan |
| [prior-art.md](prior-art.md) | References and inspirations |
