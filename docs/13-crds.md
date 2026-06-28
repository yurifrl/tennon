# Custom Resources (CRDs)

Every MCPHub object is a Kubernetes custom resource under the API group
`mcphub.io/v1alpha1`. **You never have to write this YAML** — the web UI and CLI translate
clicks and commands into these resources. The schemas are shown here for reference,
GitOps, and automation.

> How these CRDs are generated, installed, and versioned across backends is covered in
> [CRD Lifecycle](crd-lifecycle.md).

The CRD *schema* is fixed regardless of [storage backend](storage.md); on Postgres/SQLite
the same objects are persisted as rows and behave identically.

## Connectors

A **connector** is a registered upstream that MCPHub credentials, runs, and turns into a set
of callable tools. It is the credentialed, addressable unit, and whatever its type it is
served to clients over the **MCP protocol** at `/mcp/<name>`. Three concrete kinds specialize
the concept; they share the `connectors` category (`kubectl get connectors` lists them all)
but differ in how tools are produced and where they run:

| Kind | Tools come from | Runs |
|------|-----------------|------|
| **McpConnector** | discovered by connecting to an MCP server | a container / stdio process / remote endpoint |
| **OpenApiConnector** | compiled from an OpenAPI spec | in-process in the server (no Pod) |
| **GraphqlConnector** | compiled from a GraphQL endpoint | in-process |

How spec connectors are built, authenticated, versioned, and how provider-specific cases like
Google are handled is covered in [Spec Connectors](spec-connectors.md); the strategy for new
and special connectors is in [Connector Extensibility](connector-extensibility.md).

**Credentials by convention.** A connector reads a Kubernetes `Secret` named
`mcphub-<connector-name>`, assumed to already exist; the value is rendered at call time and
never exposed. Multiple accounts of the same upstream (a work and a personal Google) are
simply **two connectors** with two Secrets — there is no separate connection object.

**Who may use a connector** — and which calls require approval — is **not** part of the
connector. A connector defines only how to reach the upstream and which tools it exposes;
authorization lives entirely in org-wide [`Policy`](#policy) and identity-side
[`Role`](#role) grants, evaluated by [Authorization](authorization.md).

### McpConnector

An upstream MCP server. `spec.runtime` selects how it runs; `spec.scaling` applies only to
the container runtime.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: McpConnector
metadata:
  name: github
  namespace: mcphub
  labels:
    tags: code
spec:
  description: "GitHub repos, issues, and pull requests."   # agent-visible context
  runtime:
    container:
      image: ghcr.io/github/github-mcp-server:latest
    # stdio:  { command: npx, args: ["-y", "@modelcontextprotocol/server-github"] }
    # remote: { url: https://api.githubcopilot.com/mcp }
  transport: streamable-http
  scaling:
    replicas: 1                  # or autoscale: { minReplicas, maxReplicas, targetCPUUtilizationPercent }
  capabilityProfile: github-jail # container runtime only
  tools:
    exposePrefix: "github_"
  tags: ["code"]
status:
  phase: Ready                   # Pending | Ready | Degraded | Failed
  discoveredTools: 30
  url: /mcp/github
```

Notes:

- `runtime` is exactly one of `container`, `stdio`, `remote`, or `host` (expose a whole
  machine). Container and stdio MCPs run in a tightly jailed sandbox; stdio servers are
  bridged to Streamable HTTP by Supergateway — see [Runtime & Isolation](runtime-isolation.md).
- `scaling` is `replicas` (passthrough to the Deployment) or `autoscale` (the controller
  emits a native `HorizontalPodAutoscaler`). There is no scale-to-zero. Omitted ⇒
  `replicas: 1`.
- The credential is the `mcphub-github` Secret, rendered into request headers (remote) or
  `env` (container) at call time, never exposed to callers.

### OpenApiConnector

Tools compiled from an OpenAPI spec; the connector runs in-process (no Pod). The controller
fetches the spec, hashes it (`status.specHash`), and compiles each operation into a tool.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: OpenApiConnector
metadata:
  name: grafana-read
  namespace: mcphub
  labels:
    tags: observability
spec:
  description: "Grafana dashboards, alerts, and datasources (read-only)."
  specUrl: https://grafana.example.com/openapi.json
  baseUrl: https://grafana.example.com/api    # overrides the spec's servers; optional
  tools:
    allow: ["get_*", "search_*", "list_*"]     # scope which operations become tools
    exposePrefix: "grafana_"
  tags: ["observability", "read"]
status:
  phase: Ready
  generatedTools: 14
  specHash: "sha256:1f3c…"                     # tools cache key; changes when the spec changes
  url: /mcp/grafana-read
```

A read connector and a write connector of the same provider are two `OpenApiConnector`
objects (`grafana-read`, `grafana-write`) with two Secrets — share read access widely and
write access narrowly.

### GraphqlConnector

Tools compiled from a GraphQL endpoint via introspection; runs in-process.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: GraphqlConnector
metadata:
  name: linear
  namespace: mcphub
spec:
  description: "Linear issues, projects, and cycles."
  endpoint: https://api.linear.app/graphql
  tools:
    exposePrefix: "linear_"
status:
  phase: Ready
  generatedTools: 47
  url: /mcp/linear
```

## OAuthApp

A **registered, reusable OAuth client**. One app serves every connector whose upstream hits
the same OAuth endpoint — you register Google once, then credential personal and work
connectors against it. The client secret is a [Secrets](#secrets) reference, never inline.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: OAuthApp
metadata: { name: google-workspace, namespace: mcphub }
spec:
  authorizationUrl: https://accounts.google.com/o/oauth2/v2/auth
  tokenUrl: https://oauth2.googleapis.com/token
  clientId: "1234567890.apps.googleusercontent.com"
  clientSecretRef: { name: google-oauth, key: client_secret }
  scopes: []                          # optional default scopes; connectors may request more
  supportsDynamicRegistration: false  # RFC 7591 — auto-register a client when the provider allows it
status:
  connectors: ["google-personal", "google-work"]
```

When an upstream's OAuth endpoint supports dynamic client registration (RFC 7591) or
publishes metadata (RFC 8414 / RFC 9728), MCPHub can register the app for you; otherwise you
paste the `clientId`/secret from the provider's console. The minted access/refresh tokens are
written to the connector's Secret and renewed on expiry — they never appear in API responses,
status, logs, or the UI.

## Policy

Org-wide rules: guardrails, defaults, and per-connector sharing/grants. With connectors
carrying no access of their own, `Policy` (alongside [`Role`](#role)) is where every
grant lives — from "share the `grafana` connector with the Okta `observability` group" to
"every tool tagged `sensitive` requires approval everywhere."

Every rule — here and in a [`Role`](#role) — compiles to one
primitive: `subject × resource × when? × effect`. See
[Authorization → The rule](authorization.md#the-rule).

```yaml
apiVersion: mcphub.io/v1alpha1
kind: Policy
metadata: { name: org-guardrails }
spec:
  rules:
    # default: anyone in the org may use connectors tagged public
    - subjects:
        - kind: EmailDomain
          match: "example.com"
      resource:
        tag: public
      effect: allow
    # guardrail: every tool tagged sensitive requires approval, everywhere
    - resource:
        tag: sensitive
      effect: ask
    # guardrail: contractors can never reach internal connectors
    - subjects:
        - kind: Group
          provider: okta
          match: contractors
      resource:
        tag: internal
      effect: block
```

Global rules are evaluated together with every assigned
`Role` by the single algorithm in [Authorization → Resolution](authorization.md#resolution):
all rules are gathered and the **most-restrictive matching effect wins** (`block >
ask > allow`), so a global guardrail can only tighten, never loosen. Subject
kinds: `User`, `Group`, `EmailDomain`, `Token`, `ServiceAccount`. A rule's `resource` matches
by `connector`, `tool` (glob), or `tag`, with an optional `when` condition over the
call's arguments.

## IdentityProvider

How **humans log in**. An `IdentityProvider` registers an external OIDC provider (Google,
Okta, …) and maps its claims to MCPHub identity: it is the Teleport-style *auth connector*.
Don't confuse it with [`OAuthApp`](#oauthapp) — an `OAuthApp` is an **upstream** OAuth client
used to credential *connectors*, while an `IdentityProvider` is how *users* authenticate **to**
MCPHub. Configure these under **Admin → Identity Providers** in the UI.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: IdentityProvider
metadata: { name: okta }          # referenced by Policy `provider:`, User `identities[].provider`,
                                  # and the group-id prefix (e.g. okta:observability)
spec:
  type: oidc                       # oidc only at launch (Google & Okta are first-class)
  displayName: "Okta"
  issuer: https://acme.okta.com    # OIDC issuer; metadata discovered via RFC 8414
  clientId: "0oa1b2c3def"
  clientSecretRef: { name: okta-oidc, key: client_secret }   # never inline
  scopes: ["openid", "email", "groups"]
  claims:
    username: email                # OIDC claim used as the User name / subject
    groups: groups                 # OIDC claim carrying group membership
  claimsToRoles:                   # map claim values → MCPHub roles (Teleport's claims_to_roles)
    - { claim: groups, value: "observability",   roles: ["devtools-user"] }
    - { claim: groups, value: "sre-*",           roles: ["devtools-user"] }   # glob match on the claim value
    - { claim: groups, value: "platform-admins", roles: ["admin"] }
status:
  phase: Ready                     # Pending | Ready | Degraded | Failed
  lastLogin: "2026-06-21T09:00:00Z"
```

At login MCPHub validates the OIDC token, reads `claims.username` and `claims.groups`, and
creates (or updates) the [`User`](#user): the username becomes `metadata.name`, the claims
populate `spec.traits`, and `claimsToRoles` assigns `spec.roles`. Each `claimsToRoles.value`
matches a claim value exactly or by glob (`sre-*`); the first matching entry's roles apply.
A group from this provider
is addressed everywhere as `<provider-name>:<group-value>` — e.g. the Okta `observability`
group is `okta:observability` in a connector's `access`, a `Role`'s templating, and a `Policy`
subject (`{ kind: Group, provider: okta, match: observability }`). See
[Authentication](authentication.md) and [Users & Roles](users.md).

## User

A human identity. Mirrors Teleport's model: users are stored in the backend **and** fully
expressible as resources. **Local users** are created with `mcphub users add` or by applying
the YAML; **SSO users** are created at login from an [identity provider](authentication.md) —
their OIDC claims populate `spec.traits` and an entry under `spec.identities`. A user has no
permissions of its own; everything comes from the [`Role`](#role)s in `spec.roles`.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: User
metadata: { name: joe }
spec:
  roles: ["devtools-user"]        # assigned roles
  traits:                         # key → list-of-values, used in role templating
    email: ["joe@example.com"]
    groups: ["sre@example.com"]    # from the OIDC groups claim for SSO users
  identities:                     # SSO links (omitted for pure local users)
    - provider: google           # an IdentityProvider name
      subject: "joe@example.com"
status:
  locked: false                   # lock to instantly cut off access
  lastLogin: "2026-06-21T09:00:00Z"
```

Locking a user (`status.locked: true`, set via `mcphub users lock joe`) terminates live
sessions and blocks new ones without deleting the account. Machine identities use
[`ServiceAccount`](#serviceaccount)/[`Token`](#token) instead of `User`.

## Role

What a subject may do — both **manage** control-plane resources and **call** tools. Modeled on
Teleport's role: two condition sets, `spec.allow` and `spec.deny`, plus `spec.options`.
**A `deny` is just a rule with `effect: block`**, so "deny overrides allow" is the top of the
most-restrictive-wins order, not a separate algorithm. Roles are assigned to `User`s,
`ServiceAccount`s, and SSO groups.

```yaml
apiVersion: mcphub.io/v1alpha1
kind: Role
metadata: { name: devtools-user }
spec:
  options:
    maxSessionTtl: 8h
    lock: false                   # strict lock wins across roles
  allow:
    # Control-plane RBAC: which MCPHub resources this role may manage, and how.
    rules:
      - resources: [connector]
        verbs: [list, read]
      - resources: [token]
        verbs: [list, create, read, update, delete]
    # Data-plane: which tools this role may call.
    tools:
      - connector: github
        tools: ["*"]
      - tag: infra
        tools: ["status"]
    # Label selector over connectors (OR within a key, AND across keys; '*' = any).
    connectorLabels:
      tags: ["code", "tickets"]
  deny:
    tools:
      - tools: ["*delete*"]        # nobody with this role calls a delete tool
  # Templating: pull values from the subject's traits, e.g. grant tools by IdP group.
  # tools: [{ tag: "{{external.groups}}", tools: ["*"] }]
```

Verbs are `list, create, read, update, delete` (or `*`); resource kinds include
`connector, role, user, token, serviceAccount, identityProvider, policy, secret, capabilityProfile`.
Trait templating mirrors Teleport: `{{internal.<trait>}}` for local user traits and
`{{external.<claim>}}` for SSO claims, interpolated into `tools`, label selectors, etc. The
data-plane `tools` grants and a [`Policy`](#policy)'s rules **compile to the same rule
primitive** (see [Authorization](authorization.md#the-rule)); they differ only in who authors
them and why — a `Role` packages grants on the identity side and assigns them to subjects,
while a `Policy` holds org-wide guardrails.

**Just-in-time access** (optional): a role may grant the ability to *request* other roles for
a bounded TTL.

```yaml
spec:
  allow:
    request:
      roles: ["prod-admin"]        # may request these (wildcards ok: "prod-*")
      maxDuration: 8h
    reviewRequests:
      roles: ["prod-admin"]        # may approve/deny others' requests
```

Requests become `AccessRequest` objects that pause until a reviewer approves; on approval the
subject assumes the role only for `maxDuration`. The built-in `requester`/`reviewer` roles
provide the basic flow.

### Managing users & roles: two ways

The CRD *is* the resource: editing via `mcphub`/API/UI and applying `User`/`Role` CRs with the
Kubernetes Operator converge on one backend store. The two paths, finalizer, and
`mcphub.io/keep` retention are covered in
[Users & Roles → Two ways to manage](users.md#two-ways-to-manage-just-like-teleport).

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
      effect: allow              # allow | ask | block — same effect as a Policy/Role rule
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

Attach a profile to an `McpConnector` via `capabilityProfile: grafana-jail` (a default
profile applies if omitted). Capability profiles apply only to container/stdio McpConnectors —
`OpenApiConnector`/`GraphqlConnector` run in-process and have no container to jail. The
`status.planes` map reports, per plane, whether it is `enforced | degraded | audit |
unenforced` and which provider satisfied it — this is what the UI status line summarizes. The
`CiliumNetworkPolicy` / `KubeArmorPolicy` are controller-owned derived objects you never write
by hand.

## Secrets

On the Kubernetes backend, a connector's credential is a native `Secret` located by convention
(`mcphub-<connector-name>`) or by an explicit `secretKeyRef`. On Postgres/SQLite, MCPHub uses
an encrypted store-native secret table with the same reference shape. Either way, values are
rendered into the upstream at call time and **never** appear in API responses, status, logs,
or the UI. See [Security](security.md).
