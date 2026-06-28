# Security Model

MCPHub's job is to let people share access to powerful tools without sharing the credentials
behind them. This page summarizes the security properties and where to verify them.

## Credential isolation

- A connector's upstream credential is **referenced**, never inlined: `secretKeyRef` to a
  Kubernetes `Secret` (Kubernetes backend) or an encrypted store-native secret
  (Postgres/SQLite). Multiple credentialed variants (e.g. a read connector and a write
  connector) are separate `McpConnector` objects, each with its own credential.
- The controller **renders** the secret into the upstream MCP at call time (env vars for
  container MCPs, headers for remote MCPs), inside the cluster boundary.
- Rendered values **never** appear in API responses, resource status, logs, or the web UI.
- Callers (teammates, agents, CI) **never receive** the upstream credential. They present
  their own MCPHub identity; the proxy injects the connector's secret. This is what makes
  "share Grafana with an Okta team" safe — and lets you hand out a read connector widely
  while keeping the write connector's admin credential narrow.

## Identity & least privilege

- **Humans:** OIDC (Google/Okta), short-lived access tokens, groups from claims.
- **Machines (preferred):** tokenless trust — no stored secret; a short-lived credential is
  minted per call after validating a platform-signed document.
- **Machines (fallback):** fine-grained, short-lived tokens scoped to resources × tools ×
  level × expiry, with an org-enforceable maximum lifetime.
- Authorization is per-tool, additive, most-restrictive-wins, with allow / require-approval /
  block. See [Authorization](authorization.md).

## Short-lived by default & revocation

- Credentials are short-lived (~1h); **expiry is the primary cutoff**.
- **Revoking** a token or service account invalidates all derived credentials immediately.
- An **emergency lock** terminates in-flight sessions for a subject.

## Human-in-the-loop

Sensitive tools (by tag, connector setting, or bundle approval list) **pause** for explicit human
approval before executing. Approvals are auditable.

## Audit

Mint, revoke, policy change, connector reconcile, and every tool call (allow/deny/approve) are
recorded with the acting subject. (Audit sink configuration is deployment-specific.)

## Trust boundaries

- `mcphub-web` holds no secrets and no business logic — safe to expose publicly while the
  controller/API stays internal.
- The CLI's MCP server (`mcphub mcp serve`) is a thin local stdio bridge; it carries the
  caller's credential but performs no authorization or secret rendering — those stay in the
  controller.

## Reporting vulnerabilities

See [Contributing](contributing.md) for the security disclosure process. Do not file public
issues for vulnerabilities.
