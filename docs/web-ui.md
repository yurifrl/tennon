# Web UI

The web UI (served by `mcphub-web`, built with Vite + React) is the primary way to operate
MCPHub. It is **click-driven**: you never write YAML. Every action translates to a custom
resource behind the scenes (see [CRDs](crds.md)), and resources created out-of-band
(`kubectl`, GitOps) appear here too.

A clickable mockup of the UI ships alongside these docs — open `mockup-v2.html` (latest) or
`mockup.html` (v1). Press <kbd>⌘K</kbd> / <kbd>Ctrl K</kbd> anywhere for a command palette that
searches integrations and tools and jumps between pages.

## Sign in

OIDC only — **Google** or **Okta** (see [Authentication](authentication.md)). Machines do not
use the UI; they authenticate tokenless or with tokens.

## Layout

A collapsible sidebar (toggle with the button in the top bar) groups pages:

### Catalog
- **Dashboard** — cluster overview: integration health, bundles, active tokens, pending
  approvals, recent activity.
- **Integrations** — list and detail. Each integration shows its `kind` (mcp / openapi /
  graphql) and `runtime` (container image, **npx** package, **uvx** package, or **remote**
  endpoint) — an MCP need not ship an image. Detail shows configuration, rendered secrets
  (masked), tool exposure rules, connections, sandbox/capability status, and the exposed
  tool list. Click-driven, no YAML.
- **Tool Catalog** — search every tool across every integration by intent; inspect schemas
  and the effective policy.
- **Bundles** — the composer: pick tools from integrations, rename them, flag approvals.

### Access
- **Tokens** — list and the granular token builder (see below).
- **Service Accounts** — org-owned machine identities; tokenless trust or token.
- **OAuth Apps** — reusable OAuth clients; register a provider once, connect many accounts.
- **Secrets** — pluggable credential providers (in-cluster default, Keychain, file, 1Password,
  Vault).
- **Policies** — org-wide guardrails &amp; defaults; subjects, grants, approvals.
- **Approvals** — the human-in-the-loop queue; approve or deny paused executions.

### Admin
- **Identity Providers** — configure Google/Okta OIDC and machine trust relationships.
- **Storage** — view/select the backend (Kubernetes / Postgres / SQLite).

## The token / identity builder

The **New Token** screen offers two issuance methods:

- **Tokenless · trusted relationship (preferred):** choose a trust type (OIDC JWT, Kubernetes
  ServiceAccount, SPIFFE, cloud identity), set issuer/audience/subject. No secret is stored;
  a short-lived credential is minted per call.
- **Fine-grained token:** grant access **by MCP & tools**, **by group**, or **by tag**; set a
  short lifetime. A plain-English "What this allows" panel previews the grant.

Both translate to resources ([`ServiceAccount`](crds.md#serviceaccount) /
[`Token`](crds.md#token) / [`McpPolicy`](crds.md#mcppolicy)) — shown to you as controls, not
YAML.

## Design principle

The UI is a full control surface that *translates* CRDs, in the spirit of a click-through
product: forms, toggles, chips, and tables — never raw manifests. Power users who want YAML
can still use `kubectl`/GitOps on the Kubernetes backend; the two stay in sync.
