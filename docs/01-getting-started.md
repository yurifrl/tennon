# Getting Started

This guide gets MCPHub running and walks the flagship flow: register a credentialed MCP and
share it with a teammate — without exposing the credential.

## Prerequisites

- A place to run the controller: a Kubernetes cluster, or any host (for the Postgres/SQLite
  backends). See [Storage backends](storage.md).
- An OIDC provider for user login (Google or Okta).
- For agents: an MCP-capable client (Claude Code, Cursor, OpenCode, or Pi via a bridge).

## 1. Run the control plane

Solo / local (SQLite backend):

```bash
mcphub-controller \
  --store=sqlite --store-path=./mcphub.db \
  --api-addr=:8080 --proxy-addr=:8081 \
  --oidc-issuer=https://accounts.google.com \
  --oidc-client-id=mcphub
```

Team / cluster (Kubernetes backend) — self-contained, no repo checkout required:

```bash
# apply CRDs + controller straight from the release
kubectl apply -f https://github.com/acme/mcphub/releases/latest/download/mcphub.yaml
```

Then start the web server:

```bash
mcphub-web --listen=:3000 --api-url=http://localhost:8080 \
  --oidc-issuer=https://accounts.google.com --oidc-client-id=mcphub
```

On an empty store the controller logs a one-time **bootstrap admin token** so you can get in
before any identity provider is wired up:

```
No users found — first-run bootstrap.
  → Open:  http://localhost:3000/?token=mh_boot_8f3c1d9a7be240xZ
  → Or run: mcphub open
```

Open that URL (or run `mcphub open`), then create your first [User](users.md). The token
expires in 1h or the moment the first user exists. After that, sign in with Google or Okta.

## 2. Add a credentialed connector (Grafana)

From the web UI: **Connectors → Add connector**. Paste the Grafana MCP server URL; MCPHub
detects the transport and discovers tools. Add the credential as a secret reference (the
value is stored in the configured secret backend and never shown again).

From the CLI:

```bash
mcphub secret create grafana-creds --from-literal=token=$GRAFANA_TOKEN
mcphub connector add grafana \
  --remote https://grafana.example.com/mcp --transport streamable-http \
  --header "Authorization=Bearer {{secret:grafana-creds.token}}" \
  --expose-prefix grafana_ --tag observability
```

The credential lives with the connector. Anyone you share it with calls Grafana **through the
proxy**; they never receive the token.

## 3. Share it with an Okta team

Sharing is a **`Policy`** decision — the connector itself carries no access. From the
web UI: **Policies** → **New share**, pick the `grafana` connector, subject **Group →
Okta → `observability`**, choose which tools, save.

From the CLI (sugar that authors a `Policy` rule):

```bash
mcphub connector share grafana \
  --to "group:okta:observability" \
  --tools "grafana_get_*,grafana_search_*,grafana_list_*"
```

Membership resolves from the Okta `groups` claim at request time — people joining or leaving
the Okta team gain or lose access automatically, with no edit. Org-wide guardrails (e.g.
"`sensitive` tools always require approval") are other Policy rules and can only
tighten this grant.

## 4. Connect an agent

You add **one MCPHub URL** to your client — **no token in the config**. The first call opens a
**browser login** (the same experience Notion ships at `https://mcp.notion.com/mcp`): your
client discovers MCPHub's authorization server, registers itself (RFC 7591), runs the
authorization-code + PKCE flow against your IdP, and caches the token — reused silently on
every later call and across **every** MCP behind that URL. MCPHub injects each upstream's own
credential server-side, so you never hold it. This is the flow proven in
`hack/pocs/mcp-auth/` and documented in [Authentication](authentication.md).

Stdio clients bridge to the URL with `mcp-remote` — same shape as the Notion example:

```json
{
  "mcpServers": {
    "mcphub": {
      "type": "stdio",
      "command": "bunx",
      "args": ["-y", "mcp-remote", "https://mcphub.example.com/mcp"]
    }
  }
}
```

`/mcp` is the **global** endpoint — every connector you're allowed to use, behind
one login. Point at a single connector instead with `/mcp/grafana`.

Clients that speak MCP over Streamable HTTP natively need just the URL — no headers:

```json
{
  "mcpServers": {
    "mcphub": {
      "type": "http",
      "url": "https://mcphub.example.com/mcp"
    }
  }
}
```

**Non-interactive (CI, agents like Hermes):** there's no browser, so use
[tokenless trust](tokens-and-identity.md) (the workload's platform identity) or pass a minted
token as an `Authorization` header.

## 5. Try a tool, see an approval

Call a read tool — it runs immediately. Call a tool flagged sensitive and the execution
**pauses**; approve it from **Approvals** in the
UI (or `mcphub resume`).

## Next steps

- Mint a [granular token or set up tokenless trust](tokens-and-identity.md).
- Tune [Authorization](authorization.md).
