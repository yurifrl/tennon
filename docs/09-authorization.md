# Authorization

Authorization decides, for every call, whether the subject may proceed — and whether a human
must approve first. Tennon has two planes:

- **Control plane:** who may edit connectors, policies, roles, tokens. On the Kubernetes backend
  this is native RBAC; on Postgres/SQLite it is an equivalent role check.
  Coarse, low frequency. It lives in [`Role.rules`](users.md#roles).
- **Data plane:** who may call which tool, evaluated on **every** proxied request. Fast,
  fine-grained, and the subject of this page.

Because every [connector](crds.md) — a containerized MCP server, a compiled OpenAPI spec, a
GraphQL endpoint — is served to clients over the **MCP protocol**, the proxy sees the same
JSON-RPC on every call: the method, the tool `name`, and the `arguments`. That single,
uniform chokepoint is what lets one model gate every connector identically, down to the
argument level.

## Three outcomes

Every call resolves to exactly one of:

- **Allow** — runs without interruption.
- **Ask** — pauses for a human (see [Approvals](#approvals)).
- **Block** — cannot be called.

These are ordered by restrictiveness: `block > ask > allow`.

## The rule

Every grant, from every authoring surface, compiles to one primitive:

```
rule =
  subject     # user | group | email-domain | token | service-account
  × resource  # what it matches (see "Granularity" below)
  × when?     # optional condition over arguments / time / rate (ABAC)
  × effect    # allow | ask | block
```

There is no separate "deny" concept: a deny **is** a rule with `effect: block`. This is what
makes "deny always wins" simply the top of the restrictiveness order, not a second algorithm.

### Granularity

`resource` matches at four levels, coarse to fine. All four are MCP-native, because the proxy
is reading MCP JSON-RPC:

1. **Connector** — every tool of `grafana`.
2. **Tool glob** — `grafana/get_*`, `*/delete_*`.
3. **Exact tool** — `internal-deploy/rollback`.
4. **Tool + argument condition** (`when`) — `rollback` only when `args.env != "prod"`;
   `send_mail` only when `args.to` ends in `@acme.com`; `ask` when
   `args.amount > 1000`.

The same `resource` match also covers the other MCP objects the proxy exposes —
`resources/read` (a resource URI) and `prompts/get` (a prompt name) — so granular
permissioning is not limited to tool calls.

A worked set, increasing in precision:

```
allow    sre         */get_*, */list_*
ask      sre         internal-deploy/rollback   when args.env == "prod"
block    contractor  */delete_*
```

## Authoring surfaces

Two places author rules. They do not overlap — each has one job, and both emit the
primitive above. **Connectors carry no access of their own**; authorization is kept entirely
separate from the connector that defines how to reach an upstream.

### Policy — the default home
Most grants live in a [`Policy`](crds.md). A Policy holds both per-connector sharing and
org-wide guardrails: "share the `grafana-read` connector with the Okta `observability` group,
deletes requiring approval" is a Policy rule, just like "every tool tagged `sensitive` requires
approval everywhere." Rules name connectors, tools, or tags, so a single Policy can be as
narrow as one connector's tools or as broad as an org-wide default.

### Role — identity-side packaging
A [`Role`](users.md#roles) packages reusable grants and assigns them to subjects or personas.
It carries **both** planes: control-plane RBAC (`rules`: resources × verbs) and data-plane
tool grants. Use a Role when you manage access by persona ("devtools-user") rather than
by Policy. A Role's tool grants compile to the *same rule primitive* as a Policy — they
differ only in who authors them and why, not in what they mean.

> Rule of thumb: **grant in a Policy; package per-persona grants in a Role.**

## Resolution

One algorithm, evaluated on every call:

1. Resolve the caller's **subject and groups** (from OIDC claims, the token, or the service
   account).
2. Gather **every matching rule** for `(subject, resource)` from both surfaces: every
   `Role` assigned to the subject and all `Policy` objects (cached in-memory; watched, not
   re-read per request).
3. Evaluate each rule's `when` predicate against the live request (`arguments`, time, rate);
   drop rules whose condition fails.
4. **Most-restrictive matching effect wins:** any `block` ⇒ block; else any `ask`
   ⇒ pause; else an `allow` ⇒ allow.
5. **No matching `allow` ⇒ block.** Access is default-deny; a tool runs only because some
   rule grants it.

Because `deny` is just `effect: block`, "deny overrides allow" and "global guardrails can
only tighten" are the same step (4), not separate rules. An org-wide `Policy` can force
approval or block over an `allow` from a Role or another Policy; it can never loosen one.

## Engine: compiled to Cedar

The authoring surfaces are the human-facing layer; the controller **compiles them all to
[Cedar](https://www.cedarpolicy.com/)** (`Policy` rules and `Role` grants together) and
evaluates Cedar **in-process** on each call. This gives:

- **RBAC** — subjects → resources → tools.
- **ABAC** — the `when` predicate becomes a Cedar condition on request context (arguments,
  time, rate).
- **Analyzability** — Cedar can flag over-broad or unused grants, important when tokens grant
  tool access dynamically.

A simple compiled grant:

```
permit(
  principal in Group::"okta:observability",
  action == Action::"call",
  resource in Connector::"grafana"
);
```

An argument condition (the `when` predicate) — block destructive tools unless privileged:

```
forbid(principal, action == Action::"call", resource)
when {
  resource.tags.contains("sensitive") &&
  !principal.scopes.contains("infra:write")
};
```

### Ask as an obligation

Cedar decides permit/forbid, not a third state. Tennon models ask as a Cedar
**obligation** (advice annotation): the policy permits the call but tags it for approval, and
the proxy routes it into the approvals queue instead of forwarding.

```
@advice("ask")
permit(
  principal in Group::"sre@example.com",
  action == Action::"call",
  resource == Tool::"internal-deploy/rollback"
)
when { context.args.env == "prod" };
```

> You do not write Cedar by hand. It is a compilation target shown here for transparency.

## Approvals

When a call resolves to ask (or needs auth), the execution **pauses** and
becomes a first-class object with an id, subject, target tool, and arguments. A human
approves or denies it from **Approvals** in the [Web UI](web-ui.md), or via the API/CLI
(`tctl resume --execution-id …`). The same mechanism handles pausing for an upstream auth
step. Pending approvals have a timeout.

## Why not a relationship engine (OpenFGA/SpiceDB)?

Those excel when permissions are a deep relationship graph (org → team → connector → tool with
overrides). Tennon's needs are tool globs + group/tag membership + argument conditions, which
Cedar handles in-process with no extra stateful service — preserving the embeddable SQLite
story. If relationship inheritance becomes dominant, a relationship backend can be revisited.

## Scaling from one person to a team

For a solo user there are typically no groups: rules bind directly to the user or a single
service account. Adding a team is additive — grant a connector to a group — with no
change to the model. See [Requirements](requirements.md) (design note).
