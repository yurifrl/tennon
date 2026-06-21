# Authorization

Authorization decides, for every tool call, whether the subject may proceed — and whether a
human must approve first. MCPHub has two planes:

- **Control plane:** who may edit MCPs, bundles, policies, tokens. On the Kubernetes backend
  this is native RBAC; on Postgres/SQLite it is an equivalent role check. Coarse, low
  frequency.
- **Data plane:** who may call which tool, evaluated on **every** proxied request. Fast,
  fine-grained, additive, and supporting three outcomes.

This page covers the data plane.

## Three outcomes

Every tool call resolves to one of:

- **Allow** — runs without interruption.
- **Require approval** — pauses for a human (see [Approvals](#approvals)).
- **Block** — cannot be called.

Defaults are sensible: read-only tools tend to allow; destructive tools tend to block or
require approval. The `sensitive` tag forces approval. You can tune any tool.

## Where policies live: local vs global

Authorization comes from **two tiers**, and most policy lives at the *local* tier:

### Local policy — on the MCP / connection
Most sharing and grants live **with the thing they govern**. An `Mcp` (and each
[`McpConnection`](crds.md#mcpconnection)) carries an `access` block: which subjects may use
*this* server, which of *its* tools, and which tools require approval. This is where
day-to-day decisions live — "share the `grafana-read` connection with the Okta
`observability` group, with deletes requiring approval" is a property of that connection, set
by its owner. It stays next to the credential and tool scope it relates to.

### Global policy — org-wide general rules only
A small number of **`McpPolicy`** objects hold rules that span many MCPs: org guardrails and
defaults. Examples: "anyone in `example.com` may use `tag:public` MCPs," "every tool tagged
`sensitive` requires approval everywhere," "contractors can never touch `tag:internal`."
Global policy is intentionally *general* — it should not enumerate individual MCP grants;
those belong on the MCP.

> Rule of thumb: **grant access on the MCP/connection; set org-wide guardrails and defaults
> globally.** If a rule names one specific MCP's tools for one team, it's local.

## Resolution

The effective decision for a call combines both tiers:

1. Resolve the caller's subject and groups (from OIDC claims or the token/service account).
2. Gather the **local** `access` rules on the target MCP/connection **and** all matching
   **global** `McpPolicy` rules (cached in-memory; watched, not re-read per request).
3. Combine matches **additively** (a subject is granted if local or global grants it).
4. **Most-restrictive outcome wins** — a global `block` or `require-approval` guardrail
   overrides a local `allow`. Global guardrails can only tighten, never loosen.
5. If the resolved outcome is require-approval, pause; else allow or deny.

So an MCP owner shares their server freely at the local tier, while the org retains a global
backstop that can force approval or deny across everything.

## Engine: compiled to Cedar

Both tiers are the authoring surface; the controller **compiles them to [Cedar](https://www.cedarpolicy.com/)**
(local `access` blocks and global `McpPolicy` together) and evaluates Cedar **in-process** on
each call. This gives:

- **RBAC today** — subjects → resources → tools.
- **ABAC when needed** — conditions on request context (arguments, time, rate).
- **Analyzability** — Cedar can flag over-broad or unused grants, important when tokens
  grant tool access dynamically.

A simple compiled grant:

```
permit(
  principal in Group::"okta/observability",
  action == Action::"call",
  resource in Mcp::"grafana"
);
```

A condition (ABAC) — block destructive tools unless explicitly privileged:

```
forbid(principal, action == Action::"call", resource)
when {
  resource.tags.contains("sensitive") &&
  !principal.scopes.contains("infra:write")
};
```

### Require-approval as an obligation

Cedar decides permit/forbid, not a third state. MCPHub models require-approval as a Cedar
**obligation** (advice annotation): the policy permits the call but tags it for approval, and
the proxy routes it into the approvals queue instead of forwarding.

```
@advice("require-approval")
permit(
  principal in Group::"sre@example.com",
  action == Action::"call",
  resource == Tool::"internal-deploy/rollback"
);
```

> You do not write Cedar by hand. It is a compilation target shown here for transparency.

## Approvals

When a call resolves to require-approval (or needs auth), the execution **pauses** and
becomes a first-class object with an id, subject, target tool, and arguments. A human
approves or denies it from **Approvals** in the [Web UI](web-ui.md), or via the API/CLI
(`mcphub resume --execution-id …`). The same mechanism handles pausing for an upstream auth
step. Pending approvals have a timeout.

## Why not a relationship engine (OpenFGA/SpiceDB)?

Those excel when permissions are a deep relationship graph (org → team → bundle → tool with
overrides). MCPHub's needs are tool globs + group/tag membership, which Cedar handles
in-process with no extra stateful service — preserving the embeddable SQLite story. If
relationship inheritance becomes dominant, a relationship backend can be revisited.

## Scaling from one person to a team

For a solo user there are typically no groups: policies bind directly to the user or a single
service account. Adding a team is additive — share an MCP or bundle to a group — with no
change to the model. See [Requirements](requirements.md) (design note).
