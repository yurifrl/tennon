# MCPHub Documentation

> The Kubernetes-native control plane and proxy for Model Context Protocol (MCP) servers.

MCPHub lets one person or a whole team register MCP servers, share them safely (credentials
never leave the platform), and govern every tool call with
fine-grained authorization, short-lived tokens, and human-in-the-loop approvals — all driven
from a click-through web UI that translates to Kubernetes custom resources behind the scenes.

## Why MCPHub

- **One catalog, every agent.** Point Claude Code, Cursor, OpenCode, or Pi at MCPHub and
  share the same tools, auth, and policies across all of them.
- **Credentials stay put.** Configure an MCP that needs a credential (e.g. Grafana) once;
  share it with a teammate or an Okta group without ever handing them the secret.
- **Tokenless by default.** Machines authenticate via a trusted relationship (OIDC, K8s
  ServiceAccount, SPIFFE, cloud identity) — no shared secret to leak. Fine-grained
  short-lived tokens are the fallback.
- **Scales from solo to team.** The same model runs on SQLite for one person and on
  Postgres or Kubernetes for a team.

## Start here

- New to MCPHub? Read [Getting Started](getting-started.md).
- Want the big picture? Read [Architecture](architecture.md) and [Concepts](concepts.md).

## Documentation map

### Concepts & design
- [Concepts](concepts.md) — MCP, policy, token, service account, tags.
- [Requirements](requirements.md) — the durable constraints that shape the model.
- [Architecture](architecture.md) — the binaries, the proxy, the control loop.
- [Project Structure](project-structure.md) — repository layout.
- [Technologies](technologies.md) — the stack and why.
- [Custom Resources (CRDs)](crds.md) — `McpConnector`, `Policy`, `Token`, `ServiceAccount`.
- [CRD Lifecycle](crd-lifecycle.md) — how CRDs are generated, delivered, and evolved.
- [Spec Connectors (OpenAPI & GraphQL)](spec-connectors.md) — compiling tools from a spec; Google via Discovery.
- [Connector Extensibility](connector-extensibility.md) — how new and provider-specific connectors are supported (design note).

### Runtime & storage
- [The MCP Proxy](proxy.md) — per-connector URLs.
- [Runtime & Isolation](runtime-isolation.md) — sandboxing, Supergateway.
- [Capability Profiles](capability-profiles.md) — what an MCP container may do.
- [Capability Architecture](capability-architecture.md) — how profiles compile to enforcers (design note).
- [Storage backends](storage.md) — Kubernetes/etcd, Postgres, SQLite.

### Using MCPHub
- [Getting Started](getting-started.md)
- [Web UI](web-ui.md)
- [API](api.md)
- [CLI](cli.md)

### Access & security
- [Authentication](authentication.md) — OIDC (Google, Okta) and machine trust.
- [Authorization](authorization.md) — policies, Cedar, allow / ask / block.
- [Users & Roles](users.md) — human identity and role packaging.
- [Tokens & Identity](tokens-and-identity.md) — tokenless trust + fine-grained tokens.
- [Security model](security.md) — secret handling, revocation, audit.

### Operating
- [Deployment](deployment.md)
- [Roadmap](roadmap.md)
- [Contributing](contributing.md)
- [FAQ](faq.md)

### Appendices (design notes & research)
- [Catalog Features](features.md) — Executor-inspired catalog capabilities.
- [Repository Server](repository-server.md) — Obsidian vault as a structured API (design note).
- [Prior Art & Inspirations](prior-art.md)
- [Token & Identity — Reference](token-identity-reference.md) — research behind the token design.

## Project status

MCPHub is pre-1.0 and under active design. APIs and CRD schemas may change. See the
[Roadmap](roadmap.md) for the phased plan.

## License

Apache License 2.0.
