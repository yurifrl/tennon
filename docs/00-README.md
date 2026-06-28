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

- New to MCPHub? Read [Getting Started](getting-started.md) and the [Concepts](concepts.md).
- Building or contributing to MCPHub? The design, architecture, and internals live under
  **[Architecture](arch/_all.md)** (`/arch`).

## Documentation map

### Get started
- [Getting Started](getting-started.md) — install, first connector, connect an agent.
- [Concepts](concepts.md) — connector, tool, tag, subject, policy, token, service account.

### Use the features
- [Web UI](web-ui.md) — the click-through console.
- [CLI](cli.md) — `mcphub` from the terminal.
- [API](api.md) — the HTTP surface.
- [Spec Connectors (OpenAPI & GraphQL)](spec-connectors.md) — compile tools from a spec.
- [Capability Profiles](capability-profiles.md) — what an MCP container may do.
- [Catalog Features](features.md) — catalog capabilities.

### Access & security
- [Authentication](authentication.md) — OIDC (Google, Okta) and machine trust.
- [Authorization](authorization.md) — policies, Cedar, allow / ask / block.
- [Users & Roles](users.md) — human identity and role packaging.
- [Tokens & Identity](tokens-and-identity.md) — tokenless trust + fine-grained tokens.
- [Security model](security.md) — secret handling, revocation, audit.

### Reference & operating
- [Custom Resources (CRDs)](crds.md) — `McpConnector`, `Policy`, `Token`, `ServiceAccount`.
- [Storage backends](storage.md) — Kubernetes/etcd, Postgres, SQLite.
- [Deployment](deployment.md)
- [FAQ](faq.md)

### Architecture & internals → [`/arch`](arch/_all.md)
Requirements, architecture, project structure, technologies, CRD lifecycle, connector
extensibility, the proxy, runtime isolation, capability architecture, roadmap, contributing,
and research notes.

## Project status

MCPHub is pre-1.0 and under active design. APIs and CRD schemas may change. See the
[Roadmap](roadmap.md) for the phased plan.

## License

Apache License 2.0.
