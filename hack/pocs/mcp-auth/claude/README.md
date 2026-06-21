# MCP Auth POC — Notion-style browser login, federated to Google

An MCP server you add to any client by **URL only**. First call drives a one-time
browser login against **Google**; the resulting token is reused on later calls.

## Quick start — server + Claude Code + login

**1. Start the server** (dev login page, no Google setup needed):

```bash
npm install
JWT_SECRET=dev npm start
# -> MCP auth POC on http://localhost:8000  [DEV IdP: no Google, fake login page]
```

**2. Put this in `.mcp.json`** in your project (where you'll run Claude Code):

```json
{
  "mcpServers": {
    "auth-poc": {
      "type": "http",
      "url": "http://localhost:8000/mcp"
    }
  }
}
```

That's the whole client config — just the URL, no secret.

**3. Launch Claude Code** in that directory:

```bash
claude
```

Claude Code sees the server needs auth and prompts you — run `/mcp`, pick `auth-poc`,
choose **Authenticate**. A browser tab opens.

**4. Log in.** The browser shows a one-field sign-in page (the dev IdP standing in for
Google). Enter any email and submit. The tab closes; Claude Code is now authenticated.

**5. Use it.** Ask Claude to call the `whoami` tool — it returns
`You are authenticated as <the email you entered>`. The token is cached, so later calls
and restarts reuse it without logging in again.

To use **real Google** instead of the dev page, see [Run with real Google](#run-with-real-google) below.

## How it works

We are our **own OAuth 2.1 authorization server** (Google has no Dynamic Client
Registration), exactly like Notion's hosted MCP. MCP clients DCR + PKCE against
**us**; we federate the human login to Google and mint our own tokens.

```
client --DCR/PKCE--> us (AS)        we redirect the browser --> Google login
Google --code--> us /google/callback (we read the user's email)
us --code--> client redirect_uri    client --code+verifier--> us /token --> JWT
client --Bearer JWT--> us /mcp       whoami tool returns the email
```

The `@modelcontextprotocol/sdk` `mcpAuthRouter` provides the RFC-conformant plumbing
(RFC 9728 protected-resource metadata, RFC 8414 AS metadata, RFC 7591 DCR, RFC 6749 +
7636 authorize/token with mandatory PKCE S256). `server.js` only implements the
`OAuthServerProvider` hooks plus the Google callback. Tokens are HS256 JWTs (`jose`):
access ~1h, refresh 30d, rotated on refresh.

## Run locally — no Google needed (dev IdP)

If `GOOGLE_CLIENT_ID` is unset the server runs a built-in **fake login page** instead of
Google. The entire OAuth flow (DCR, PKCE, tokens, `/mcp`, token reuse) is identical — only
the IdP is stubbed, so you prove the hypothesis with zero external setup.

```bash
npm install
JWT_SECRET=dev npm start          # prints "[DEV IdP: no Google, fake login page]"
node test_flow.mjs                # scripts a full MCP client: DCR -> login -> whoami x2
```

Add `http://localhost:8000/mcp` to an MCP client; the browser opens a one-field email form
rather than Google, then `whoami` returns that email.

## Run with real Google

1. Create a Google OAuth **Web application** client. Authorized redirect URI:
   `http://localhost:8000/google/callback`. Scopes used: `openid email`.
2. `cp .env.example .env` and fill `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `JWT_SECRET`.
3. `env $(grep -v '^#' .env | xargs) npm start`
4. Add `http://localhost:8000/mcp` to an MCP client (Claude, Cursor, `mcp-remote`, ...).
   The client discovers the AS, self-registers, opens the Google login, then calls `whoami`.

## Verify without a client

```bash
curl -s localhost:8000/.well-known/oauth-protected-resource/mcp   # RFC 9728
curl -s localhost:8000/.well-known/oauth-authorization-server     # RFC 8414
curl -i -X POST localhost:8000/mcp -d '{}'   # 401 + WWW-Authenticate resource_metadata
node test_tokens.mjs                          # token contract self-check
node test_flow.mjs                            # full end-to-end flow via the dev IdP
```

## Deploy (optional)

Only if local browser login is insufficient. Follow the GitOps pattern in
`/Users/yuri/Workdir/Yuri/board-games` (GH Actions -> GHCR, Helm chart, ArgoCD app +
istio VirtualService for a public host). Set `PUBLIC_URL` to the public host and add
`${PUBLIC_URL}/google/callback` to the Google client. Secrets via External Secrets / 1Password.

## Acceptance criteria status

- [x] (1) Unauthenticated `/mcp` -> 401 pointing at the AS (verified locally)
- [x] (2) Both well-known docs resolve and name the AS + endpoints (verified locally)
- [x] (3) Full browser login with no user-side client registration — `test_flow.mjs` (dev IdP) / Google
- [x] (4) `whoami` returns the logged-in user's email — `test_flow.mjs`
- [x] (5) Same token accepted on later calls — `test_flow.mjs` (call 2 reuses the token)

R3 (login against the *real* Google IdP) needs a Google OAuth client; the dev IdP proves
every other requirement end-to-end without it.

Skipped: RFC 8707 resource indicators (parsed but not enforced), server-side refresh
storage, refresh reuse-detection. Add when moving past POC.
