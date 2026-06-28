# Roadmap

A phased plan. Phases 1-3 establish the Kubernetes-native proxy; phases 4-5 add the
catalog, approvals, and the UI. Pre-1.0: APIs and CRD schemas may change.

## Phase 1 — Controller + per-MCP proxy
The end-to-end goal: **register an MCP, attach it to an agent, and let the agent call its
tools using the connector's credentials** — no credential ever exposed to the caller.

- `McpConnector` CRD and reconciler (container + remote runtimes).
- State-backed routing (no single config file).
- Proxy with per-MCP URLs and tool prefixing/allow-deny.
- Credential rendered from the connector's Secret into the upstream call at request time.
- **Done when:** applying an `McpConnector` spawns/connects the server and proxies its tools at
  `/mcp/<name>` with the configured prefix, and an agent calls those tools using the
  connector's stored credential.

## Phase 1.1 — Spec connectors (integrations)
Same end-to-end goal as Phase 1, but for **spec connectors** (`OpenApiConnector` /
`GraphqlConnector`) that compile tools from a spec and run in-process. **Google is the proof
of concept:** an `OpenApiConnector` fed by Google's API Discovery documents, authenticated via
an `OAuthApp`.

- Spec fetch → compile → serve at `/mcp/<name>`, identical to an MCP connector downstream.
- `OAuthApp` + minted token rendered into the call; multiple accounts = multiple connectors.
- **Done when:** a Google `OpenApiConnector` is registered, attached to an agent, and the agent
  calls a Gmail/Calendar tool using the OAuth credential. See
  [Spec Connectors](spec-connectors.md).

## Phase 2 — Storage backends & authentication
- Pluggable store: Kubernetes/etcd, Postgres, SQLite (same CRD types).
- Secret rendering from references (in-cluster only).
- OIDC login (Google, Okta); service tokens.
- **Done when:** the same objects work on all three backends; upstream credentials never
  leak; a user logs in via OIDC and a CI token authenticates.

## Phase 3 — Authorization
- `Policy` CRD compiled to Cedar, evaluated in-process.
- Watched, cached policies; additive, most-restrictive-wins; allow/ask/block.
- **Done when:** two subjects see different allowed tools; policy edits take effect live.

## Phase 4 — Tokenless identity & catalog
- Tokenless trust: OIDC JWT, Kubernetes ServiceAccount, SPIFFE, cloud identity.
- Fine-grained tokens with tags as the sharing primitive; org max-lifetime.
- Unified tool catalog; source auto-detection; intent search.
- **Done when:** a workload (e.g. Hermes) calls tools with no stored secret; adding a source
  auto-detects transport and indexes tools.

## Phase 5 — Approvals & web UI
- Pause/resume executions (human-in-the-loop) for sensitive tools.
- Click-driven Vite/React UI: catalog, policy builder, token/identity
  builder, approvals inbox, identity-provider and storage admin.
- CLI with embedded MCP server for programmatic AI agent access.
- **Done when:** a sensitive call pauses and waits for approval; the full lifecycle (add
  source → policy → call → approve) is doable from the UI; AI agents can connect
  to the CLI's MCP server to perform operations programmatically.

## Beyond
- Workflows, per-execution scratch storage, richer audit/usage analytics, and additional
  trust types as needed. See the design notes ([requirements](requirements.md),
  [repository server](repository-server.md), [token & identity](token-identity-reference.md)).
