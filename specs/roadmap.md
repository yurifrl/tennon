# MCPHub Roadmap

A phased plan. Phases 1-3 establish the Kubernetes-native proxy; phases 4-6 add the
executor-inspired catalog, approvals, and UI.

## Phase 1: Controller + per-MCP proxy

### Features
- `Mcp` CRD and reconciler (container + remote runtimes).
- State-backed routing table (no single config file).
- MCP proxy with per-MCP URLs (`/mcp/<name>`).
- Tool prefixing / allow-deny from the `Mcp` spec.

### Success Criteria
- Apply an `Mcp` CR and have its server spawned/connected.
- Discover and proxy that server's tools at `/mcp/<name>`.
- Tools appear to clients with the configured prefix.

## Phase 2: Secrets & authentication

### Features
- Secret rendering from `secretKeyRef` into env / headers (in-cluster only).
- OAuth2 (authorization code + PKCE) for user authentication.
- Service tokens for machine clients (create / scope / revoke).

### Success Criteria
- Upstream credentials never appear in API responses, status, or logs.
- A user can log in via OAuth2 and reach the proxy.
- A scoped service token authenticates a CI client.

## Phase 3: Authorization

### Features
- `McpPolicy` CRD: subjects (groups, email domains, service tokens) → allowed MCPs/tools.
- Watched, in-memory policy cache; additive evaluation.
- `403` on denied tool calls.

### Success Criteria
- Two users with different policies see different allowed tools.
- Policy edits take effect live without restart.

## Phase 4: Bundles

### Features
- `McpBundle` CRD: curate + rename tools across multiple `Mcp`s.
- Bundle endpoints (`/bundle/<name>`) with their own auth boundary.
- Bundle-level routing back to source MCPs.

### Success Criteria
- A bundle aggregates tools from 2+ MCPs behind one URL.
- Renamed tool names route correctly to upstream tools.

## Phase 5: Catalog & discovery

### Features
- Unified tool catalog from all `Mcp` resources.
- Source auto-detection (transport probe + initial discovery) on create.
- Intent-based tool search (API + proxy `discover` meta-tool).

### Success Criteria
- Adding a source from the UI auto-detects transport and indexes tools.
- Natural-language search returns relevant tools across the catalog.

## Phase 6: Approvals & web UI

### Features
- Pause/resume executions (human-in-the-loop) for sensitive tools.
- Approvals inbox, catalog browser, bundle composer, policy/token management — Vite/React UI.
- Audit & usage views.

### Success Criteria
- A sensitive tool call pauses and waits for UI/API approval.
- The full lifecycle (add source → bundle → policy → call → approve) is doable from the UI.

## Technical Stack

### `mcphub-controller`
- Go, Kubebuilder / controller-runtime.
- HTTP API + MCP proxy data plane in one process.
- OAuth2 validation + service-token issuance.

### `mcphub-web`
- Vite + React (TypeScript).
- Serves the SPA and reverse-proxies to the controller API.
