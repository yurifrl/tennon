# MCPHub — Core Requirements

Durable, must-not-forget requirements that shape the architecture and authorization model.

## R1 — Scales from one person to a whole team

The same model must work for a single solo user and for an entire team, with no fork in
behavior. Solo usage is just the degenerate case (one subject, no shared groups). Adding a
team should not require a different product, a different deployment, or a rewrite of how
authorization works.

Implications:
- The pluggable storage backends cover the range: SQLite/local for one person, Postgres or
  Kubernetes/etcd for a team.
- The authorization model (subjects = user / group / service token) must read naturally for
  one user and for many. Groups are optional, not mandatory.
- Sharing is additive: a solo setup becomes a team setup by sharing an existing MCP, not by
  reconfiguring it.

## R2 — Set up a credentialed MCP and share it with an IdP team (Okta)

The headline use case:

> As the owner, I configure an MCP that needs credentials (for example **Grafana**, with an
> API token or service-account credential), and I **share it with the members of a team
> defined in my identity provider (Okta)** — without ever exposing the underlying
> credential to those members.

Requirements this implies:
- **Credential ownership stays with the owner.** The Grafana credential is referenced by
  the `Mcp` (secret reference, rendered in-cluster / in the secret store), never handed to
  the people it's shared with. Shared users call the MCP through the proxy; the proxy
  injects the owner's credential. Members never see or hold it.
- **Sharing targets an IdP group, not individuals.** I grant access to the Okta team/group
  (e.g. `grafana-viewers@example.com` or an Okta group id), and membership is resolved from
  the OIDC `groups` claim at request time. People joining/leaving the Okta team gain/lose
  access automatically, with no policy edit.
- **Identity provider is pluggable OIDC.** Google and **Okta** must both be supported as
  OAuth2/OIDC providers; group/team membership comes from the provider's claims. (The
  earlier "Google-only" sketch is superseded: it is OIDC, with Okta as a first-class
  provider for team identity and groups.)
- **Granular sharing.** Sharing can be the whole MCP, a subset of its tools, or a bundle —
  scoped to the group, and optionally gated by approval for sensitive tools.

### Example shape

```yaml
# The owner's credentialed MCP
kind: Mcp
metadata:
  name: grafana
spec:
  runtime: { type: remote, url: https://grafana.example.com/mcp }
  transport: streamable-http
  headers:
    - name: Authorization
      valueFrom:
        secretKeyRef: { name: grafana-creds, key: token, format: "Bearer {{ .value }}" }
  tools:
    exposePrefix: "grafana_"
---
# Share it with an Okta team — members never see the credential
kind: McpPolicy
metadata:
  name: share-grafana-with-observability
spec:
  subjects:
    - kind: Group
      provider: okta
      match: "observability"        # Okta group; resolved from OIDC groups claim
  rules:
    - mcp: grafana
      tools: ["grafana_get_*", "grafana_search_*", "grafana_list_*"]
```

In Cedar terms this becomes a `permit(principal in Group::"okta/observability", action ==
Action::"call", resource in Mcp::"grafana")` with the owner's credential resolved by the
proxy at call time. See [authentication-flow.md](authentication-flow.md) and the
authorization design.
