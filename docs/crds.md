# Custom Resources (CRDs)

Every MCPHub object is a Kubernetes custom resource under the API group
`mcphub.io/v1alpha1`. **You never have to write this YAML** — the web UI and CLI translate
clicks and commands into these resources. The schemas are shown here for reference,
GitOps, and automation.

The CRD *schema* is fixed regardless of [storage backend](storage.md); on Postgres/SQLite
the same objects are persisted as rows and behave identically.

## Integration

The **definition** of an upstream tool provider: how to reach it and where its tool catalog
comes from. An MCP server is one `kind` of integration; OpenAPI and GraphQL services are
others. Tools are **rebuilt from the integration's spec** — connections, credentials, and
policies are preserved across a rebuild. An `Integration` is credential-agnostic on its own;
credentials live on a [`Connection`](#connection). For convenience it may declare one inline
`defaultConnection` so single-credential setups stay a single object.

`spec.kind` selects the source plugin and which `spec.source` shape applies:

```yaml
# kind: mcp — an upstream MCP server
apiVersion: mcphub.io/v1alpha1
kind: Integration
metadata:
  name: grafana
  namespace: mcphub
  labels:
    tags: "observability"
spec:
  kind: mcp                    # mcp | openapi | graphql
  description: "Grafana dashboards, alerts, and datasources."  # agent-visible context
  source:
    runtime:
      type: remote             # container | remote
      url: https://grafana.example.com/mcp
    transport: streamable-http # streamable-http only (stdio servers are bridged by Supergateway)
  tools:
    exposePrefix: "grafana_"    # catalog-level prefix
  # Optional: inline default connection for the simple one-credential case.
  defaultConnection:
    secretKeyRef: { name: grafana-creds, key: token, as: header, format: "Bearer {{ .value }}" }
status:
  phase: Ready                 # Pending | Ready | Degraded | Failed
  discoveredTools: 30
  specHash: "sha256:1f3c…"     # tools cache key; changes when the spec changes
  connections: ["grafana-read", "grafana-write"]
```

```yaml
# kind: openapi — an OpenAPI service, or a bundle of several (e.g. Google)
apiVersion: mcphub.io/v1alpha1
kind: Integration
metadata: { name: google, namespace: mcphub, labels: { tags: "productivity" } }
spec:
  kind: openapi
  description: "Bundle Gmail, Calendar, Drive, Docs, and other Google APIs into one source."
  source:
    # A single spec URL, or several discovery docs bundled into one integration.
    discoveryUrls:
      - https://www.googleapis.com/discovery/v1/apis/gmail/v1/rest
      - https://www.googleapis.com/discovery/v1/apis/calendar/v3/rest
      - https://www.googleapis.com/discovery/v1/apis/drive/v3/rest
    refetchOnSave: false       # re-pull the spec(s) and rebuild tools on the next apply
status:
  phase: Ready
  specHash: "sha256:9ab0…"
  discoveredTools: 214
```

```yaml
# kind: graphql — a GraphQL endpoint (tools from introspection)
apiVersion: mcphub.io/v1alpha1
kind: Integration
metadata: { name: linear, namespace: mcphub }
spec:
  kind: graphql
  description: "Linear issues, projects, and cycles."
  source:
    endpoint: https://api.linear.app/graphql
status: { phase: Ready, discoveredTools: 47 }
```

For `kind: mcp`, container sources set `source.runtime.type: container` with an `image`
(credentials rendered as `env`); remote sources render into request `headers`. Container MCPs
run in a tightly jailed sandbox and stdio servers are bridged to Streamable HTTP by
Supergateway — see [Runtime & Isolation](runtime-isolation.md). For `kind: openapi`/`graphql`
the controller fetches the spec, hashes it (`status.specHash`), and compiles each operation
into a tool; the spec text is stored by hash, never inline. In all cases secret values are
resolved in-cluster and never exposed.

The **preset catalog** (Google, Stripe, GitHub REST, Vercel, …) is not stored as resources —
it is a static library the UI/CLI seed from. Picking a preset just produces an `Integration`
of the right `kind` with the spec URL(s) and suggested OAuth config prefilled.

## Connection

A named, **credentialed instance** of an `Integration`. This is how you use the same provider
with different credentials and tool scopes — e.g. a read connection and a write connection of
one Grafana MCP, or a personal and a work account of the Google integration. Each MCP
connection gets its own proxied URL (`/mcp/<name>`); OpenAPI/GraphQL connections expose their
tools into the catalog and [bundles](bundles.md). Connections are independently shared and
governed.

```yaml
# API-key / header credential (resolved from a Secrets provider)
apiVersion: mcphub.io/v1alpha1
kind: Connection
metadata: { name: grafana-read, namespace: mcphub }
spec:
  integrationRef: grafana
  identityLabel: "Grafana (read-only)"
  credential:
    secretKeyRef: { name: grafana-readonly, key: token, as: header, format: "Bearer {{ .value }}" }
  tools:
    allow: ["get_*", "search_*", "list_*"]
  access:                      # local policy — who may use THIS connection (lives with it)
    - subject: { group: "okta:observability" }
      tools: ["*"]             # of this connection's exposed tools
    - subject: { token: "ci-runner" }
      tools: ["get_*"]
  approvals:
    require: ["delete_*"]
  tags: ["observability", "read"]
status:
  url: /mcp/grafana-read
  exposedTools: 14
---
# OAuth credential (minted via a shared OAuthApp; token stored in a Secrets provider)
apiVersion: mcphub.io/v1alpha1
kind: Connection
metadata: { name: google-work, namespace: mcphub }
spec:
  integrationRef: google
  identityLabel: "Google (work account)"
  oauth:
    appRef: google-workspace   # an OAuthApp targeting Google's OAuth endpoint
    scope: "https://mail.google.com/ https://www.googleapis.com/auth/calendar"
  tools:
    allow: ["gmail_*", "calendar_*"]
  tags: ["productivity"]
status:
  exposedTools: 38
  expiresAt: "2026-06-21T12:00:00Z"   # OAuth token expiry; re-minted on demand
```

Notes:

- `integrationRef` points at the `Integration` definition (shared discovery + spec).
- A connection carries **either** a `credential` (api-key/header, resolved from a Secrets
  provider) **or** an `oauth` block referencing a reusable [`OAuthApp`](#oauthapp).
- `tools.allow/deny` scopes which of the integration's tools this connection exposes.
- **`access` is the connection's local policy** — who may use it and which of its tools, plus
  `approvals`. Sharing lives here (e.g. "give the Okta `observability` group read access"),
  next to the credential it relates to, set by the connection's owner. Most policy is local.
- Org-wide guardrails/defaults live separately in a global [`McpPolicy`](#mcppolicy); they can
  only tighten a connection's local grants (force approval / deny), never loosen them.
- Tokens and bundles also target a **connection** (e.g. `connection: grafana-read`),
  so you can share read access widely and write access narrowly.
- The simple case (`Integration.defaultConnection`) creates an implicit connection named
  after the integration, addressable at `/mcp/<integration>` for MCP kinds.

A Repo's driver and attached MCPs (see the Repository Server design note) are connections
too, which is how read vs read-write access to a shared filesystem is expressed.

## OAuthApp

A **registered, reusable OAuth client**. One app serves every `Connection` whose integration
hits the same OAuth endpoint — you register Google once, then connect personal and work
accounts against it. The client secret is a [Secrets](#secrets) reference, never inline.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: OAuthApp
metadata: { name: google-workspace, namespace: mcphub }
spec:
  authorizationUrl: https://accounts.google.com/o/oauth2/v2/auth
  tokenUrl: https://oauth2.googleapis.com/token
  clientId: "1234567890.apps.googleusercontent.com"
  clientSecretRef: { name: google-oauth, key: client_secret }
  scopes: []                          # optional default scopes; connections may request more
  supportsDynamicRegistration: false  # RFC 7591 — auto-register a client when the provider allows it
status:
  connections: ["google-personal", "google-work"]
```

When an integration's OAuth endpoint supports dynamic client registration (RFC 7591) or
publishes metadata (RFC 8414 / RFC 9728), MCPHub can register the app for you; otherwise you
paste the `clientId`/secret from the provider's console. The minted access/refresh tokens are
written to the connection's Secrets provider and renewed on expiry — they never appear in API
responses, status, logs, or the UI.

## McpBundle

A curated, renamed set of tools from one or more **connections** (of any integration kind)
behind a single endpoint.

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

## McpPolicy (global)

Org-wide **general** rules only — guardrails and defaults that span many connections. Per-MCP
sharing and grants do **not** go here; they live in each connection's `access` block (above).
Global policy should stay general: domain/tag defaults, mandatory approvals, blanket denies.
See [Authorization](authorization.md).

```yaml
apiVersion: mcphub.io/v1alpha1
kind: McpPolicy
metadata: { name: org-guardrails }
spec:
  scope: global
  rules:
    # default: anyone in the org may use connections tagged public
    - subjects: [{ kind: EmailDomain, match: "example.com" }]
      where: { tag: public }
      grant: { tools: ["*"] }
    # guardrail: every tool tagged sensitive requires approval, everywhere
    - where: { tag: sensitive }
      effect: require-approval
    # guardrail: contractors can never reach internal connections
    - subjects: [{ kind: Group, provider: okta, match: "contractors" }]
      where: { tag: internal }
      effect: block
```

Global rules are evaluated together with each connection's local `access` (see
[Authorization → Resolution](authorization.md)): grants are additive, and a global
`require-approval`/`block` **overrides** a local allow — guardrails only tighten. Subject
kinds: `User`, `Group`, `EmailDomain`, `ServiceToken`, `ServiceAccount`. A rule's `where`
matches by `tag`, `integration`, `connection`, or `bundle`.

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

## CapabilityProfile

Declares, in plain intent, **what an MCP container is allowed to do** (network, files,
processes, sandbox). Default-deny: anything not listed is blocked. The controller compiles
it to enforcer-native objects depending on what the cluster has — starting with **Cilium**
for egress, KubeArmor for process/file, and gVisor for the sandbox. Users author this via the
UI/CLI; see the user guide [Sandbox & Access](capability-profiles.md) and the
[design doc](capability-architecture.md).

```yaml
apiVersion: mcphub.io/v1alpha1
kind: CapabilityProfile
metadata: { name: grafana-jail, namespace: mcphub }
spec:
  baseline: model-armor        # hardened default to start from
  default: deny                # anything not listed is blocked
  sandbox: strong              # strong (gVisor) | maximal (Kata) | off
  allow:
    network:                   # domains it may reach; compiled to Cilium FQDN egress
      - { host: grafana.example.com, port: 443 }
      - { dns: true }
    files:
      - { path: /etc/grafana, mode: read }
      - { repo: notes, path: /vault, mode: read-write }
    run:                       # processes allowed to exec (defaults to the server)
      - /usr/bin/supergateway
      - /app/grafana-mcp
  capabilities: { drop: ["ALL"] }
  resources: { cpu: "500m", memory: 256Mi, pids: 128 }
  enforce: require             # require (fail closed) | best-effort
status:
  posture: require
  planes:
    network: { state: enforced, by: cilium }
    process: { state: enforced, by: kubearmor }
    file:    { state: enforced, by: kubearmor }
    sandbox: { state: enforced, by: gvisor }
  derived:
    ciliumNetworkPolicy: grafana-jail-net
    kubeArmorPolicy:     grafana-jail-ka
```

Attach a profile to an integration or connection via `runtime.capabilityProfile: grafana-jail` (a
default profile applies if omitted). The `status.planes` map reports, per plane, whether it
is `enforced | degraded | audit | unenforced` and which provider satisfied it — this is what
the UI status line summarizes. The `CiliumNetworkPolicy` / `KubeArmorPolicy` are
controller-owned derived objects you never write by hand.

## Secrets

On the Kubernetes backend, secrets are native `Secret` objects referenced by `secretKeyRef`.
On Postgres/SQLite, MCPHub uses an encrypted store-native secret table with the same
reference shape. Either way, values are rendered into the upstream MCP at call time and
**never** appear in API responses, status, logs, or the UI. See [Security](security.md).
