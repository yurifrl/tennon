# Web UI

The web UI (served by `mcphub-web`, built with Vite + React) is the primary way to operate
MCPHub. It is **click-driven**: you never write YAML. Every action translates to a custom
resource behind the scenes (see [CRDs](crds.md)), and resources created out-of-band
(`kubectl`, GitOps) appear here too.

A clickable mockup of the UI ships alongside these docs — open `mockup.html`.

## Sign in

OIDC only — **Google** or **Okta** (see [Authentication](authentication.md)). Machines do not
use the UI; they authenticate tokenless or with tokens.

## Layout

A collapsible sidebar (toggle with the button in the top bar) groups pages:

### Catalog
- **Dashboard** — cluster overview: MCP health, bundles, active tokens, pending approvals,
  recent activity.
- **MCPs** — list and detail. Detail shows configuration (runtime, transport, prefix, tags),
  rendered secrets (masked), tool exposure rules, and the exposed tool list. Click-driven,
  no YAML.
- **Bundles** — the composer: pick tools from MCPs, rename them, flag approvals.
- **Tool Catalog** — search every tool across every MCP by intent; inspect schemas.

### Access
- **Tokens** — list and the granular token builder (see below).
- **Service Accounts** — org-owned machine identities; tokenless trust or token.
- **Policies** — the visual policy builder: subjects, grants, approvals.
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
