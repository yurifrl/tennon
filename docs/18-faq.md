# FAQ

### How is Tennon different from connecting an MCP server directly?

A direct connection means per-server config in every client, credentials often sitting
client-side, and no central authorization. Tennon gives you one per-connector endpoint, central [authentication](authentication.md) and
[authorization](authorization.md), credentials that stay server-side, and human approvals on
sensitive tools. See [The MCP Proxy](proxy.md).

### Do I have to use Kubernetes?

No. The [storage backend](storage.md) is pluggable. Run on **SQLite** for one person,
**Postgres** for a team without a cluster, or **Kubernetes/etcd** for the native experience.
The objects and behavior are identical across backends.

### Do I have to write YAML?

No. The [web UI](web-ui.md) and [CLI](cli.md) are the intended interfaces and translate your
actions into custom resources. On the Kubernetes backend you *can* also use `kubectl`/GitOps;
the UI and CRDs stay in sync.

### Can I share an MCP that needs a credential without giving the credential away?

Yes — this is the core use case. The credential is owned by the [`TennonConnector`](crds.md#tennonconnector) and
rendered into the upstream server at call time. People you share with call the MCP through
the proxy and never receive the secret. See [Getting Started](getting-started.md) and
[Security](security.md).

### How do I share with a team?

Grant a [policy](authorization.md) to a **group** from your identity provider (e.g. an Okta
group). Membership resolves from the OIDC `groups` claim at request time, so people joining or
leaving the team gain or lose access automatically.

### What identity providers are supported?

OIDC, with **Google** and **Okta** as first-class providers for human login. Machines use
[tokenless trust](tokens-and-identity.md) (OIDC JWT, Kubernetes ServiceAccount, SPIFFE, cloud
identity) or fine-grained tokens.

### What does "tokenless" mean?

The machine proves its identity with a platform-signed document (e.g. a GitHub Actions OIDC
token or a Kubernetes ServiceAccount token) instead of a stored secret. Tennon validates it
against a trust rule and mints a short-lived credential per call. Nothing to leak. See
[Tokens & Identity](tokens-and-identity.md).

### Does it work for just me?

Yes. Solo is the degenerate case: one subject, no groups, SQLite store. Adding a team later is
additive — you share existing connectors to a group — with no reconfiguration.

### Which agents can connect?

Any MCP-capable client (Claude Code, Cursor, OpenCode). Agents connect **directly** to a
`/mcp/...` URL over Streamable HTTP, or — for clients that launch a stdio
command — via the CLI's MCP server (`tctl mcp serve`).

### What happens when a tool needs approval?

The execution pauses and appears in the **Approvals** queue. A human approves or denies it in
the UI (or via `tctl resume`), then it continues or is rejected. See
[Authorization → Approvals](authorization.md#approvals).

### Why Cedar for policies?

[Cedar](https://www.cedarpolicy.com/) is an open-source policy language and authorization
engine (from AWS). It gives RBAC now, ABAC (conditions) later, and analyzability to catch
over-broad grants — all in-process, with no extra stateful service, preserving the embeddable
SQLite story. You author [`Policy`](crds.md#policy); Cedar is the compilation target. See
[Authorization](authorization.md).

### Is it production ready?

Not yet — Tennon is pre-1.0 and under active design. See the [Roadmap](roadmap.md).
