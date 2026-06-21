# SPEC: an authenticated MCP URL, like Notion

> **This is a proof of concept.** Quick and easy, no over-engineering. We are testing one
> hypothesis: *can we give an MCP URL a Notion-style browser login?* Use **Google** as the
> identity provider. Build the least that proves the hypothesis and stop — anything beyond
> that is out of scope.

## 1. What we want

An MCP server we host at **our own URL** that a user adds to any MCP client and authenticates
with a **one-time browser login** — exactly the experience Notion ships at
`https://mcp.notion.com/mcp`. One login, no secret pasted into client config, token reused
across calls and across the MCPs behind that URL.

## 2. What that looks like — Notion's auth, technically

Notion is the reference. Its hosted MCP server is the thing we are cloning the *behaviour* of,
so the spec describes its components precisely.

**The server.** `https://mcp.notion.com/mcp` is a hosted, remote **Streamable-HTTP** MCP
server (legacy SSE at `/sse`) that acts as an **OAuth 2.1 protected resource**. It is its
**own authorization server** — it does not delegate the OAuth surface to a third party. It is
built on Cloudflare's `workers-oauth-provider`. The hosted server accepts **OAuth only** —
classic Notion integration/PAT bearer tokens are rejected.

**Discovery + auth chain (the components):**

1. **Protected Resource Metadata — RFC 9728.** An unauthenticated request can be answered
   with `401` + `WWW-Authenticate: ... resource_metadata="…"`; clients may also fetch
   `/.well-known/oauth-protected-resource` directly. Its `authorization_servers` names
   Notion's own AS. (Notion's docs mislabel this "RFC 9470" — that is an error; the standard
   is RFC 9728.)
2. **Authorization Server Metadata — RFC 8414.** `/.well-known/oauth-authorization-server`
   advertises `authorization_endpoint`, `token_endpoint`, `registration_endpoint`, and
   `code_challenge_methods_supported: [S256]`.
3. **Dynamic Client Registration — RFC 7591.** Supported. Clients `POST` to the
   `registration_endpoint` as a public client (`token_endpoint_auth_method: "none"`) and
   self-register with **no pre-shared secret** — this is what makes "just add the URL" work.
4. **Authorization Code + PKCE — RFC 6749 / 7636.** Mandatory PKCE (S256).
5. **Resource Indicators — RFC 8707.** Recommended by the MCP spec; Notion's support is
   unverified (see gaps).

**Human consent.** The browser opens Notion's authorization endpoint; the user logs into
their Notion account and selects which **workspace** to authorize. Grants are
**workspace-scoped per user**, with per-user credential isolation. This is a distinct OAuth
surface from Notion's classic public/internal integration OAuth on `api.notion.com`.

**Tokens.** Bearer access token on every `/mcp` call, **~1h expiry**. Refresh token with a
**30-day absolute (non-sliding) max**, **rotated on every refresh** (reuse of a rotated token
revokes the whole grant; `invalid_grant` is terminal → re-auth).

**Cross-client UX.** Claude, Cursor, ChatGPT, VS Code, `mcp-remote` all run the identical
flow: one browser login → DCR + PKCE → tokens cached in the client's secure store → reused
silently until refresh/expiry.

*Sources: developers.notion.com/guides/mcp; modelcontextprotocol.io specification/authorization;
RFC 9728 / 8414 / 7591 / 7636 / 8707.*

## 3. Goal

Stand up **our** equivalent: an authenticated MCP URL that a user adds to a client, logs into
**once via a browser against a real IdP**, and then makes authorized tool calls — matching the
Notion components above as closely as a POC needs to. Prove it works end to end.

## 4. Requirements

- **R0 — Streamable HTTP, not SSE.** The server speaks MCP over **Streamable HTTP**. SSE is
  not used anywhere (matches MCPHub: "MCP over streamable-HTTP; no SSE").
- **R1 — URL-only client config.** A user adds only our URL; no secret in the client config.
- **R0 — Streamable HTTP, not SSE.** The server speaks MCP over **Streamable HTTP**. SSE is
  not used anywhere (matches MCPHub: "MCP over streamable-HTTP; no SSE").
- **R2 — Browser login, then reuse.** First call drives an interactive browser login; the
  resulting token is reused on later calls without re-login.
- **R3 — Real IdP (Google).** Login authenticates against **Google** and resolves to a real
  user identity (at minimum, their email).
- **R4 — Standards-conformant discovery.** Protected-resource metadata (RFC 9728) +
  authorization-server metadata (RFC 8414), OAuth 2.1 authorization-code + PKCE (S256). A
  standard MCP client discovers and connects with **no manual per-client registration by the
  user** — via DCR (RFC 7591) or an equivalent that hides registration from the user.
- **R5 — One login across MCPs.** A single login yields a domain-scoped token reusable across
  every MCP/bundle reached through that URL — not one login per MCP.
- **R6 — Verified on each call.** Every call presents the token; the server verifies validity
  and expiry and rejects invalid/expired tokens.

## 5. Validation & deployment

**Local first.** The implementer SHOULD validate the whole flow locally (the server, the
discovery docs, a browser login, a token-bearing tool call).

**Cluster is optional, not required.** Browser-login OAuth needs a public HTTPS URL for the
redirect, which may be impossible to validate purely locally. **If, and only if, local
validation is not sufficient, the implementer CAN deploy to the cluster** to get a real public
URL. They are not required to.

If deploying, follow the existing GitOps pattern, of which `/Users/yuri/Workdir/Yuri/board-games`
is the worked example:

- A **GitHub Actions** workflow builds the container and pushes it to **GHCR**
  (`board-games/.github/workflows/build-and-push.yml`).
- A **Helm chart** in the app repo renders the workload
  (`board-games/chart/` — Deployment/Service/PVC/ConfigMap).
- An **ArgoCD `Application`** in `/Users/yuri/Workdir/Yuri/home-systems/k8s/applications/`
  (see `board-games.yaml`) syncs the chart, and the cluster `support` chart attaches an istio
  **VirtualService** giving a public host (e.g. `bg.syscd.live`). That public host is the URL
  the MCP client and the OAuth redirect use.

Deploy secrets come from the secret store (External Secrets / 1Password), never the repo.

## 6. Acceptance criteria

Done when all hold (locally, or against the deployed public URL if the cluster path is used):

1. An unauthenticated request to `/mcp` points the client at the authorization server (R4).
2. `/.well-known/oauth-protected-resource` and `/.well-known/oauth-authorization-server`
   resolve and name the AS + its endpoints (R4).
3. A fresh client given only the URL completes a browser login against the real IdP with no
   user-performed client registration (R1–R4).
4. After login, a demo tool call succeeds and returns the logged-in user's real identity (R3,
   R6).
5. The same token is accepted on a later call without re-login (R2, R5).

## 7. Open questions & gaps

- Which **Google** federation approach satisfies R4 with the least code, given Google lacks
  DCR (RFC 7591)?
- RFC 8707 resource indicators — required for this POC, or deferred?
- Where do refresh tokens live for the POC — client cache only, or server-side?
- Notion's exact AS hostname, advertised scopes, and RFC 8707 support are **unverified**
  (well-known JSON was not directly fetchable during research).

## 8. References

- Unauthenticated starting point: `../stdio-mcp-bridge/` (Dockerfile, k8s.yaml).
- Deploy pattern example: `/Users/yuri/Workdir/Yuri/board-games` (+ its ArgoCD app in
  `/Users/yuri/Workdir/Yuri/home-systems/k8s`).
- Production model this POC validates: `docs/authentication.md`.
