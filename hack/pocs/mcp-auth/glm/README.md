# mcp-auth-glm

POC for an authenticated MCP URL with a Notion-style browser login — the server
is its own OAuth 2.1 authorization server and delegates the human login to
**Google**. See `../SPEC.md`.

## What it is

- A **Streamable-HTTP** MCP server at `/mcp` (no SSE transport).
- Its own **OAuth 2.1 AS**: RFC 9728 protected-resource metadata + RFC 8414 AS
  metadata, RFC 7591 dynamic client registration (public clients, no secret),
  authorization-code + PKCE (S256), RFC 8707 resource indicator passed through.
- `/authorize` 302s to Google; `/callback` verifies Google's ID token and issues
  our own access token (JWT, ~1h) + refresh token (30d, rotated; reuse revokes
  the grant, like Notion).
- One login yields a **domain-scoped** token reused across every call (R5).
- Every `/mcp` call verifies the bearer token (R6); an `echo`/`whoami` tool
  returns the real logged-in email (R3).

## Run

```bash
npm install
GOOGLE_CLIENT_ID=... GOOGLE_CLIENT_SECRET=... \
ISSUER=http://localhost:8787 \
GOOGLE_REDIRECT_URI=http://localhost:8787/callback \
npm start
```

Register a Google OAuth client (Cloud Console → APIs & Services → Credentials →
OAuth client ID, type "Web application") with an authorized redirect URI of
`http://localhost:8787/callback`. Google allows loopback `http://localhost`
redirects, so local validation needs no tunnel.

Point any MCP client that speaks OAuth 2.1 (Claude Desktop, Cursor, `mcp-remote`,
the SDK client) at `http://localhost:8787/mcp`. The client does discovery → DCR →
browser login → token → tool call, with no secret pasted into client config.

## Use it from Claude Code

This repo ships the wiring so you can drive the whole flow from
`hack/pocs/mcp-auth/`:

1. `cd hack/pocs/mcp-auth` and start the server with the Taskfile task:

   ```bash
   task run:glm
   ```

   That runs `npm install` + `npm start` in `glm/` and serves `/mcp` on
   `:8787`. Leave it running in its own terminal.

2. The `.mcp.json` next to the Taskfile already points Claude Code only at the
   URL — no secret, no client id, nothing else (R1):

   ```json
   { "mcpServers": { "glm": { "url": "http://localhost:8787/mcp" } } }
   ```

3. From the same `hack/pocs/mcp-auth/` dir, launch Claude Code (`claude` in the
   shell). It reads the local `.mcp.json`.
4. On the first tool call from the `glm` server, Claude Code hits `/mcp`, gets the
   `401 + WWW-Authenticate: resource_metadata=...`, pulls the discovery docs,
   runs DCR, and opens your browser at `/authorize` → Google login → consent →
   back to `/callback` → `/token` (R2–R4, acceptance #3). Tokens land in Claude
   Code's secure store; nothing is pasted anywhere.
5. Say "use the glm echo tool to say hello" — Claude Code calls `tools/call` with
   the cached bearer; the server verifies it and returns your real email
   (acceptance #4). Later calls reuse the same token with no re-login (R5–R6).

That's the whole loop: add one URL, log in once in the browser, make authorized
tool calls. That's the hypothesis the POC set out to prove.

For the real Google login, the Taskfile task auto-loads the Google credentials
from `mcphub/.env` (`task run:glm` sources `{{.TASKFILE_DIR}}/../../../.env`),
so just running the task is enough — no manual env vars. To skip Google and
exercise the rest of the flow, start the stub-IdP variant instead so the browser
opens a fake `/authorize`:

```bash
TEST_STUB_IDP=1 GOOGLE_CLIENT_ID=test-client task run:glm
```

The redirect URI Google sees is `http://localhost:8787/callback` (Google allows
loopback `http://localhost` redirects, so no tunnel is needed). Register that
URI in the Google OAuth client (Cloud Console → Credentials → the client) if it
isn't already.

### Quick local smoke (no Google, no Claude Code)

```bash
task run:glm &          # or: TEST_STUB_IDP=1 GOOGLE_CLIENT_ID=test-client task run:glm &
# in glm/:  node check.mjs     # exercises discovery, 401, DCR, PKCE, token, refresh rotation + reuse
```

`TEST_STUB_IDP=1` short-circuits `/authorize` to issue a code for
`tester@example.com` without Google, so `check.mjs` can exercise every code path
except the real IdP identity. The Claude Code path above still works against a
stub-IdP server — you'll be "logged in" as `tester@example.com` after the
browser opens the (stubbed) `/authorize`.

## Deploying to the cluster (optional, only if local isn't enough)

Browser-login OAuth needs a public HTTPS redirect for some clients; if local
loopback isn't sufficient, follow the existing GitOps pattern
(`/Users/yuri/Workdir/Yuri/board-games` is the worked example):

1. **GitHub Actions** builds the container and pushes to **GHCR**
   (see `board-games/.github/workflows/build-and-push.yml`).
2. **Helm chart** renders the workload
   (`board-gees/chart/` — Deployment/Service/ConfigMap).
3. **ArgoCD `Application`** in `/Users/yuri/Workdir/Yuri/home-systems/k8s/applications/`
   syncs the chart; the cluster `support` chart attaches an istio VirtualService
   giving a public host (e.g. `mcp.syscd.live`).

Set `ISSUER=https://<public-host>` and `GOOGLE_REDIRECT_URI=https://<public-host>/oauth/callback`,
register that redirect URI in the Google client, and source
`GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` from the secret store (External
Secrets / 1Password) — never the repo.

A `Dockerfile` is included mirroring `../stdio-mcp-bridge/Dockerfile`.

## Files

- `server.js` — the whole thing: HTTP, discovery, DCR, authorize/callback, token,
  protected `/mcp`, MCP tools.
- `check.mjs` — self-check for the non-Google paths.

## Notes / out of scope (POC)

- In-memory state (clients, codes, tokens, refresh grants). Restarts invalidate
  everything. Adequate for proving the hypothesis.
- HS256 signing key is random per process. Tokens don't survive restart.
- No per-user workspace selection (Notion's workspace picker) — one identity per
  login. Auto-consent on the first Google login.
- RFC 8707 resource indicators are accepted on `/authorize`/`/token` but not
  bound into the access token beyond the `scope` claim.
