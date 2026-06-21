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

## 2. Add a credentialed MCP (Grafana)

From the web UI: **MCPs → Add MCP**. Paste the Grafana MCP URL; MCPHub detects the transport
and discovers tools. Add the credential as a secret reference (the value is stored in the
configured secret backend and never shown again).

From the CLI:

```bash
mcphub secret create grafana-creds --from-literal=token=$GRAFANA_TOKEN
mcphub mcp add grafana \
  --remote https://grafana.example.com/mcp --transport streamable-http \
  --header "Authorization=Bearer {{secret:grafana-creds.token}}" \
  --expose-prefix grafana_ --tag observability
```

The credential lives with the MCP. Anyone you share it with calls Grafana **through the
proxy**; they never receive the token.

## 3. Share it with an Okta team

Sharing is a **local** decision on the connection — not a separate global policy. From the
web UI: open the `grafana` MCP → **Access & sharing** → **Share**, pick subject **Group →
Okta → `observability`**, choose which tools, save.

From the CLI:

```bash
mcphub connection share grafana-read \
  --to "group:okta:observability" \
  --tools "grafana_get_*,grafana_search_*,grafana_list_*"
```

Membership resolves from the Okta `groups` claim at request time — people joining or leaving
the Okta team gain or lose access automatically, with no edit. Org-wide guardrails (e.g.
"`sensitive` tools always require approval") live separately in Global Policies and can only
tighten this grant.

## 4. Connect an agent

Tokenless (preferred, e.g. for CI/Hermes): configure a [trusted relationship](tokens-and-identity.md)
and let the workload authenticate with its platform identity.

With a token, via the CLI's MCP server (for clients that launch a stdio command):

```jsonc
{
  "mcpServers": {
    "mcphub": {
      "command": "mcphub",
      "args": ["mcp", "serve", "--transport", "stdio"],
      "env": {
        "MCPHUB_URL": "https://mcphub.example.com",
        "MCPHUB_TOKEN": "mh_svc_…"
      }
    }
  }
}
```

Or point directly at a proxied endpoint over HTTP:

```jsonc
{
  "mcpServers": {
    "mcphub-grafana": {
      "type": "streamable-http",
      "url": "https://mcphub.example.com/mcp/grafana",
      "headers": { "Authorization": "Bearer <oidc-or-service-token>" }
    }
  }
}
```

## 5. Try a tool, see an approval

Call a read tool — it runs immediately. Call a tool flagged sensitive (or one under a
bundle's approval list) and the execution **pauses**; approve it from **Approvals** in the
UI (or `mcphub resume`).

## Next steps

- Compose a [Bundle](bundles.md) to give a team exactly the tools they need.
- Mint a [granular token or set up tokenless trust](tokens-and-identity.md).
- Tune [Authorization](authorization.md).
