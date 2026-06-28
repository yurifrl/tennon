# Roadmap

A phased plan. Phases 1-3 establish the Kubernetes-native proxy; phases 4-6 add bundles, the
catalog, approvals, and the UI. Pre-1.0: APIs and CRD schemas may change.

## Phase 1 — Controller + per-MCP proxy
- `McpConnector` CRD and reconciler (container + remote runtimes).
- State-backed routing (no single config file).
- Proxy with per-MCP URLs and tool prefixing/allow-deny.
- **Done when:** applying an `McpConnector` spawns/connects the server and proxies its tools at
  `/mcp/<name>` with the configured prefix.

## Phase 2 — Storage backends & authentication
- Pluggable store: Kubernetes/etcd, Postgres, SQLite (same CRD types).
- Secret rendering from references (in-cluster only).
- OIDC login (Google, Okta); service tokens.
- **Done when:** the same objects work on all three backends; upstream credentials never
  leak; a user logs in via OIDC and a CI token authenticates.

## Phase 3 — Authorization
- `Policy` CRD compiled to Cedar, evaluated in-process.
- Watched, cached policies; additive, most-restrictive-wins; allow/deny/require-approval.
- **Done when:** two subjects see different allowed tools; policy edits take effect live.

## Phase 4 — Bundles
- `Bundle` CRD: curate/rename tools across connectors; bundle endpoints with their own auth.
- **Done when:** a bundle aggregates 2+ connectors behind one URL and renamed tools route
  correctly.

## Phase 5 — Tokenless identity & catalog
- Tokenless trust: OIDC JWT, Kubernetes ServiceAccount, SPIFFE, cloud identity.
- Fine-grained tokens with tags as the sharing primitive; org max-lifetime.
- Unified tool catalog; source auto-detection; intent search.
- **Done when:** a workload (e.g. Hermes) calls tools with no stored secret; adding a source
  auto-detects transport and indexes tools.

## Phase 6 — Approvals & web UI
- Pause/resume executions (human-in-the-loop) for sensitive tools.
- Click-driven Vite/React UI: catalog, bundle composer, policy builder, token/identity
  builder, approvals inbox, identity-provider and storage admin.
- CLI with embedded MCP server for programmatic AI agent access.
- **Done when:** a sensitive call pauses and waits for approval; the full lifecycle (add
  source → bundle → policy → call → approve) is doable from the UI; AI agents can connect
  to the CLI's MCP server to perform operations programmatically.

## Beyond
- Workflows, per-execution scratch storage, richer audit/usage analytics, and additional
  trust types as needed. See the design notes ([requirements](requirements.md),
  [repository server](repository-server.md), [token & identity](token-identity-reference.md)).
