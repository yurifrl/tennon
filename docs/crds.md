# Custom Resources (CRDs)

Every MCPHub object is a Kubernetes custom resource under the API group
`mcphub.io/v1alpha1`. **You never have to write this YAML** — the web UI and CLI translate
clicks and commands into these resources. The schemas are shown here for reference,
GitOps, and automation.

The CRD *schema* is fixed regardless of [storage backend](storage.md); on Postgres/SQLite
the same objects are persisted as rows and behave identically.

## Mcp

An upstream MCP server MCPHub manages and proxies.
## Mcp

The **definition** of an upstream MCP server: how to run/reach it and its tool catalog. An
`Mcp` is credential-agnostic on its own; credentials live on an
[`McpConnection`](#mcpconnection). For convenience an `Mcp` may declare one inline
`defaultConnection` so single-credential setups stay a single object.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: Mcp
metadata:
  name: grafana
  namespace: mcphub
  labels:
    tags: "observability"
spec:
  runtime:
    type: remote               # container | remote
    url: https://grafana.example.com/mcp
  transport: streamable-http   # streamable-http only (stdio servers are bridged by Supergateway)
  tools:
    exposePrefix: "grafana_"    # catalog-level prefix
  # Optional: inline default connection for the simple one-credential case.
  defaultConnection:
    secretKeyRef: { name: grafana-creds, key: token, as: header, format: "Bearer {{ .value }}" }
status:
  phase: Ready                 # Pending | Ready | Degraded | Failed
  discoveredTools: 30
  connections: ["grafana-read", "grafana-write"]
```

Container MCPs set `runtime.type: container` with an `image`; credentials are rendered as
`env`. Remote MCPs render into request `headers`. In both cases secret values are resolved
in-cluster and never exposed. Container MCPs run in a tightly jailed sandbox and stdio
servers are bridged to Streamable HTTP by Supergateway — see
[Runtime & Isolation](runtime-isolation.md).

## McpConnection

A named, **credentialed instance** of an `Mcp`. This is how you source the same server with
different credentials and tool scopes — e.g. a read connection and a write connection of one
Grafana MCP. Each connection gets its own proxied URL (`/mcp/<name>`) and is independently
shared and governed.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: McpConnection
metadata: { name: grafana-read, namespace: mcphub }
spec:
  mcpRef: grafana
  identityLabel: "Grafana (read-only)"
  credential:
    secretKeyRef: { name: grafana-readonly, key: token, as: header, format: "Bearer {{ .value }}" }
  tools:
    allow: ["get_*", "search_*", "list_*"]
  tags: ["observability", "read"]
status:
  url: /mcp/grafana-read
  exposedTools: 14
---
apiVersion: mcphub.io/v1alpha1
kind: McpConnection
metadata: { name: grafana-write, namespace: mcphub }
spec:
  mcpRef: grafana
  identityLabel: "Grafana (admin)"
  credential:
    secretKeyRef: { name: grafana-admin, key: token, as: header, format: "Bearer {{ .value }}" }
  tools:
    allow: ["*"]
  approvals:
    require: ["create_*", "update_*", "delete_*"]
  tags: ["observability", "write"]
status:
  url: /mcp/grafana-write
  exposedTools: 30
```

Notes:

- `mcpRef` points at the `Mcp` definition (shared discovery + runtime).
- `credential` is the per-connection secret reference (rendered in-cluster, never exposed).
- `tools.allow/deny` scopes which of the MCP's tools this connection exposes; `approvals`
  can gate writes.
- Policies, tokens, and bundles target a **connection** (e.g. `connection: grafana-read`),
  so you can share read access widely and write access narrowly.
- The simple case (`Mcp.defaultConnection`) creates an implicit connection named after the
  MCP, addressable at `/mcp/<mcp>`.

A Repo's driver and attached MCPs (see the Repository Server design note) are connections
too, which is how read vs read-write access to a shared filesystem is expressed.

## McpBundle

A curated, renamed set of tools from one or more MCP **connections** behind a single
endpoint.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: McpBundle
metadata: { name: devtools, namespace: mcphub }
spec:
  tools:
    - from: github               # a connection name
      select: ["search_*", "get_*", "list_*"]
      as: "gh_{{ .tool }}"
    - from: jira
      select: ["create_issue", "get_issue", "search"]
      as: "jira_{{ .tool }}"
    - from: internal-deploy
      select: ["status", "rollback"]
  approvals:
    require: ["jira_create_issue", "rollback"]
status:
  url: /bundle/devtools
  exposedTools: 9
```

See [Bundles](bundles.md).

## McpPolicy

Maps subjects to what they may do. See [Authorization](authorization.md).

```yaml
apiVersion: mcphub.io/v1alpha1
kind: McpPolicy
metadata: { name: sre-team }
spec:
  subjects:
    - kind: Group
      provider: okta             # google | okta
      match: "observability"     # resolved from OIDC groups claim
    - kind: ServiceToken
      match: "ci-runner"
  rules:
    - bundle: devtools
      tools: ["*"]
    - tag: infra
      tools: ["status", "rollback"]
    - connection: grafana-read
      tools: ["*"]
  approvals:
    require: ["rollback"]
```

Subject kinds: `User`, `Group`, `EmailDomain`, `ServiceToken`, `ServiceAccount`. Rule
targets: `connection`, `mcp` (all connections of a definition), `bundle`, or `tag`, each with
a `tools` glob list. Targeting `connection: grafana-read` vs `connection: grafana-write` is
how you grant read access broadly and write access narrowly.

## ServiceAccount

An org-owned machine identity. Prefer a trusted relationship over a stored secret.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: ServiceAccount
metadata: { name: hermes }
spec:
  owner: platform-team
  trust:                         # tokenless — no shared secret
    type: oidc                   # oidc | kubernetes | spiffe | aws | gcp | azure
    issuer: "https://token.actions.githubusercontent.com"
    audience: "mcphub"
    subject: "repo:acme/hermes:ref:refs/heads/main"
  credential:
    ttl: 1h
    renewable: false
```

See [Tokens & Identity](tokens-and-identity.md).

## Token

A fine-grained, short-lived credential bound to a subject. Used when tokenless is not
possible.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: Token
metadata: { name: support-bot }
spec:
  subject: "service-account:support-bot"
  grants:
    - tag: chat
      tools: ["slack_post_message", "slack_read_*"]
  ttl: 1h                        # short-lived; re-mint on expiry
  maxLifetime: 90d               # org-enforceable ceiling
status:
  lastFour: "2a71"
  expiresAt: "2026-06-20T12:00:00Z"
```

The token secret is shown once at creation and never stored in plaintext.

## Secrets

On the Kubernetes backend, secrets are native `Secret` objects referenced by `secretKeyRef`.
On Postgres/SQLite, MCPHub uses an encrypted store-native secret table with the same
reference shape. Either way, values are rendered into the upstream MCP at call time and
**never** appear in API responses, status, logs, or the UI. See [Security](security.md).
