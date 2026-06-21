# MCP Auth POC

Proof of concept: Google OAuth-protected MCP server (Streamable HTTP), like Notion.

## Quick Start

### 1. Start the server

```bash
cd hack/pocs/mcp-auth/kimi
npm run dev          # Demo mode (fake auth, no Google creds needed)
```

### 2. Configure Claude Code

Add to your `.mcp.json` (in your home directory or project root):

```json
{
  "mcpServers": {
    "auth-poc": {
      "url": "http://localhost:8000/mcp"
    }
  }
}
```

### 3. Launch Claude Code and authenticate

```bash
claude
```

When Claude tries to connect, it will:
1. Get a 401 from `/mcp` with resource metadata
2. Discover the OAuth endpoints
3. Open your browser for login
4. (In demo mode, this auto-approves as `demo@example.com`)

Once authenticated, try:

```
> Use the whoami tool
```

Claude should respond: "Authenticated as: demo@example.com"

---

## Automated Tests

```bash
npm test
```

## Full Flow Test
```bash
node -e '
const http = require("http");

// 1. Register client
const register = (cb) => {
  const req = http.request({
    hostname: "localhost", port: 8000, path: "/oauth/register",
    method: "POST", headers: { "Content-Type": "application/json" }
  }, res => { let d=""; res.on("data",c=>d+=c); res.on("end",()=>cb(JSON.parse(d))); });
  req.write(JSON.stringify({redirect_uris:["http://localhost/callback"]}));
  req.end();
};

register(client => {
  console.log("Client:", client.client_id);
  console.log("Authorize: http://localhost:8000/oauth/authorize?response_type=code&client_id=" + client.client_id + "&redirect_uri=http://localhost/callback&code_challenge=test123&code_challenge_method=S256");
});
'
```

## How It Works

1. Client registers via `POST /oauth/register` (RFC 7591)
2. Client redirects user to `/oauth/authorize` with PKCE (RFC 7636)
3. Server redirects to Google (or auto-approves in demo mode)
4. User completes Google login, callback returns auth code
5. Client exchanges code for token via `POST /oauth/token`
6. Client calls `/mcp` with Bearer token

## Spec Compliance

- **RFC 9728**: Protected Resource Metadata (`/.well-known/oauth-protected-resource`)
- **RFC 8414**: Authorization Server Metadata (`/.well-known/oauth-authorization-server`)
- **RFC 7591**: Dynamic Client Registration (`/oauth/register`)
- **RFC 6749/7636**: Authorization Code + PKCE (`/oauth/authorize`, `/oauth/token`)
- **MCP Streamable HTTP**: `POST /mcp` with `Accept: application/json, text/event-stream`

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/.well-known/oauth-protected-resource` | GET | RFC 9728 metadata |
| `/.well-known/oauth-authorization-server` | GET | RFC 8414 metadata |
| `/oauth/register` | POST | Client registration |
| `/oauth/authorize` | GET | Auth endpoint (redirects to Google) |
| `/oauth/callback` | GET | Google OAuth callback |
| `/oauth/token` | POST | Token exchange |
| `/mcp` | POST | Protected MCP endpoint |

## Configuration

Set these for real Google OAuth:

```bash
export GOOGLE_CLIENT_ID=your_client_id
export GOOGLE_CLIENT_SECRET=your_client_secret
export BASE_URL=https://your-domain.com  # for production
```

Without these, runs in demo mode with auto-approved `demo@example.com` user.

## Architecture

`server.js` - Single file containing:
- In-memory stores (clients, tokens, pending codes)
- OAuth 2.1 flow implementation
- MCP request handling with one `whoami` tool
- Streamable HTTP response format

## Acceptance Criteria

| AC | Description | Status |
|---|---|---|
| 1 | Unauthenticated `/mcp` returns 401 with resource metadata | ✓ |
| 2 | Well-known endpoints resolve correctly | ✓ |
| 3 | Browser login flow works | ✓ (manual test) |
| 4 | Tool call returns user identity | ✓ |
| 5 | Token reuse works | ✓ |
