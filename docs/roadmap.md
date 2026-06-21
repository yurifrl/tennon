# Roadmap

A phased plan. Phases 1-3 establish the Kubernetes-native proxy; phases 4-6 add bundles, the
catalog, approvals, and the UI. Pre-1.0: APIs and CRD schemas may change.

## Phase 1 — Controller + per-MCP proxy
- `Mcp` CRD and reconciler (container + remote runtimes).
- State-backed routing (no single config file).
- Proxy with per-MCP URLs and tool prefixing/allow-deny.
- **Done when:** applying an `Mcp` spawns/connects the server and proxies its tools at
  `/mcp/<name>` with the configured prefix.

## Phase 2 — Storage backends & authentication
- Pluggable store: Kubernetes/etcd, Postgres, SQLite (same CRD types).
- Secret rendering from references (in-cluster only).
- OIDC login (Google, Okta); service tokens.
- **Done when:** the same objects work on all three backends; upstream credentials never
  leak; a user logs in via OIDC and a CI token authenticates.

## Phase 3 — Authorization
- `McpPolicy` CRD compiled to Cedar, evaluated in-process.
- Watched, cached policies; additive, most-restrictive-wins; allow/deny/require-approval.
- **Done when:** two subjects see different allowed tools; policy edits take effect live.

## Phase 4 — Bundles
- `McpBundle` CRD: curate/rename tools across MCPs; bundle endpoints with their own auth.
- **Done when:** a bundle aggregates 2+ MCPs behind one URL and renamed tools route
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
- **Done when:** a sensitive call pauses and waits for approval; the full lifecycle (add
  source → bundle → policy → call → approve) is doable from the UI.

## Beyond
- Workflows, per-execution scratch storage, richer audit/usage analytics, and additional
  trust types as needed. See design notes under [`specs/`](../specs/).
