// Authenticated MCP URL, Notion-style. Streamable HTTP MCP server that is its own
// OAuth 2.1 authorization server, delegating the human login to Google.
//
//   client --(1)--> /mcp                 -> 401 + WWW-Authenticate: resource_metadata
//   client --(2)--> /.well-known/...     -> discovery (RFC 9728 + 8414)
//   client --(3)--> /register            -> DCR (RFC 7591), public client, no secret
//   client --(4)--> /authorize           -> 302 to Google -> /callback -> 302 back with ?code
//   client --(5)--> /token               -> access_token (JWT, ~1h) + refresh_token (30d, rotated)
//   client --(6)--> /mcp w/ Bearer       -> verified, tool call returns the real user email
//
// One login yields a domain-scoped token reused across every call (R2, R5).

import express from "express";
import crypto from "node:crypto";
import { OAuth2Client } from "google-auth-library";
import * as jose from "jose";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { z } from "zod";

const PORT = Number(process.env.PORT ?? 8787);
// Public base URL the MCP client and Google redirect see. For local validation
// Google accepts http://localhost loopback redirects; in the cluster this is the
// public https host (e.g. https://mcp.syscd.live).
const ISSUER = (process.env.ISSUER ?? `http://localhost:${PORT}`).replace(/\/$/, "");
const GOOGLE_CLIENT_ID = process.env.GOOGLE_CLIENT_ID ?? "";
const GOOGLE_CLIENT_SECRET = process.env.GOOGLE_CLIENT_SECRET ?? "";
const GOOGLE_REDIRECT_URI = process.env.GOOGLE_REDIRECT_URI ?? `${ISSUER}/callback`;

const ACCESS_TTL = 60 * 60; // 1h, like Notion
const REFRESH_TTL = 30 * 24 * 60 * 60; // 30d absolute, non-sliding
const CODE_TTL = 10 * 60; // auth code lifetime
const LOGIN_STATE_TTL = 10 * 60;

// ponytail: in-memory everything. Single-process POC. Restarts invalidate all
// state (clients, codes, tokens) — fine for proving the hypothesis, not for prod.
const clients = new Map(); // client_id -> { redirect_uris, grant_types, ... }
const authCodes = new Map(); // code -> { client_id, redirect_uri, code_challenge, email, scope, exp }
const loginStates = new Map(); // state -> { oauth params, exp }
const refreshGrants = new Map(); // jti -> { grantId, active }; grantId -> Set<jti>

// HS256 signing key for our own access/refresh tokens. Random per process.
const secret = crypto.createSecretKey(crypto.randomBytes(32));

const app = express();
app.use(express.json());

// --- helpers --------------------------------------------------------------

const rnd = (n = 32) => crypto.randomBytes(n).toString("base64url");
const b64url = (s) => Buffer.from(s).toString("base64url");
const sha256 = (s) => crypto.createHash("sha256").update(s).digest();
const now = () => Math.floor(Date.now() / 1000);
const pick = (m, k) => { const v = m.get(k); m.delete(k); return v; };

async function signJwt(claims, ttl) {
  return new jose.SignJWT(claims)
    .setProtectedHeader({ alg: "HS256" })
    .setIssuedAt()
    .setExpirationTime(`${ttl}s`)
    .sign(secret);
}

async function verifyJwt(token) {
  const { payload } = await jose.jwtVerify(token, secret);
  return payload;
}

async function issueTokens({ email, scope, grantId }) {
  // Access token: stateless JWT, verified per call by signature + exp only.
  // One login → one token reused across every MCP behind this URL (R5): the
  // token carries no per-MCP binding, only the domain identity.
  const grant = grantId ?? rnd(8);
  const access = await signJwt(
    { sub: email, email, scope, typ: "access", grant },
    ACCESS_TTL,
  );
  const refreshJti = rnd(8);
  const refresh = await signJwt(
    { sub: email, email, scope, typ: "refresh", grant, jti: refreshJti },
    REFRESH_TTL,
  );
  if (!refreshGrants.has(`__grant__${grant}`)) refreshGrants.set(`__grant__${grant}`, new Set());
  refreshGrants.get(`__grant__${grant}`).add(refreshJti);
  refreshGrants.set(refreshJti, { grantId: grant, active: true });
  // ponytail: map keyed by refresh jti; grant revocation = deactivate all jtis in grant set.
  return { access_token: access, refresh_token: refresh, token_type: "Bearer", expires_in: ACCESS_TTL, scope };
}

// Rotate: old jti becomes inactive, new one issued. Reuse of an inactive jti
// revokes the whole grant (Notion's "reuse of a rotated token revokes the grant").
async function rotateRefresh(oldToken, scope) {
  const payload = await verifyJwt(oldToken);
  if (payload.typ !== "refresh") throw oauthError("invalid_grant", "not a refresh token");
  const entry = refreshGrants.get(payload.jti);
  if (!entry || !entry.active) {
    revokeGrant(payload.grant); // reuse → revoke everything
    throw oauthError("invalid_grant", "refresh token revoked");
  }
  entry.active = false;
  return issueTokens({ email: payload.email, scope: scope ?? payload.scope, grantId: payload.grant });
}

function revokeGrant(grantId) {
  const grant = refreshGrants.get(`__grant__${grantId}`);
  if (!grant) return;
  for (const jti of grant) {
    const e = refreshGrants.get(jti);
    if (e) e.active = false;
  }
}

function oauthError(code, desc) {
  const e = new Error(`${code}: ${desc}`);
  e.oauth = { code, desc };
  return e;
}

// --- discovery (RFC 9728 + 8414) -----------------------------------------

app.get("/.well-known/oauth-protected-resource", (_req, res) => {
  res.json({
    resource: `${ISSUER}/mcp`,
    authorization_servers: [ISSUER],
    bearer_methods_supported: ["header"],
    scopes_supported: ["openid", "email", "mcp"],
    resource_documentation: `${ISSUER}/.well-known/oauth-protected-resource`,
  });
});

app.get("/.well-known/oauth-authorization-server", (_req, res) => {
  res.json({
    issuer: ISSUER,
    authorization_endpoint: `${ISSUER}/authorize`,
    token_endpoint: `${ISSUER}/token`,
    registration_endpoint: `${ISSUER}/register`,
    grant_types_supported: ["authorization_code", "refresh_token"],
    response_types_supported: ["code"],
    token_endpoint_auth_methods_supported: ["none"],
    code_challenge_methods_supported: ["S256"],
    scopes_supported: ["openid", "email", "mcp"],
    revocation_endpoint: `${ISSUER}/revoke`,
  });
});

// --- DCR (RFC 7591) -------------------------------------------------------

app.post("/register", (req, res) => {
  const { redirect_uris = [], grant_types = ["authorization_code"], response_types = ["code"], token_endpoint_auth_method = "none", scope = "openid email mcp" } = req.body ?? {};
  if (token_endpoint_auth_method !== "none") {
    return res.status(400).json({ error: "unsupported_token_endpoint_auth_method" });
  }
  const client_id = rnd(16);
  clients.set(client_id, { client_id, redirect_uris, grant_types, response_types, token_endpoint_auth_method, scope });
  res.status(201).json({
    client_id,
    client_id_issued_at: now(),
    redirect_uris,
    grant_types,
    response_types,
    token_endpoint_auth_method: "none",
    scope,
  });
});

// --- authorization endpoint (browser login against Google) ----------------

function googleClient() {
  if (!GOOGLE_CLIENT_ID) throw new Error("GOOGLE_CLIENT_ID not set");
  return new OAuth2Client(GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URI);
}

app.get("/authorize", (req, res) => {
  const { client_id, redirect_uri, response_type, code_challenge, code_challenge_method, state, scope } = req.query;
  // Accept any client_id: clients cache the DCR'd id across our (in-memory)
  // restarts, and some skip DCR entirely. Public clients carry no secret, so the
  // id is just a correlation handle — auto-register on first sight.
  let client = clients.get(client_id);
  if (!client) {
    client = { client_id, redirect_uris: redirect_uri ? [redirect_uri] : [], grant_types: ["authorization_code"], response_types: ["code"], token_endpoint_auth_method: "none", scope: "openid email mcp" };
    clients.set(client_id, client);
  }
  if (response_type !== "code") return res.status(400).send("unsupported response_type");
  if (code_challenge_method !== "S256") return res.status(400).send("PKCE S256 required");
  if (redirect_uri && client.redirect_uris.length && !client.redirect_uris.includes(redirect_uri)) {
    client.redirect_uris.push(redirect_uri);
  }
  // TEST_STUB_IDP=1: skip Google and mint an auth code directly. Lets the
  // self-check (check.mjs) exercise PKCE/token/refresh/reuse with no live IdP.
  // The real Google identity path is /authorize -> Google -> /callback below.
  if (process.env.TEST_STUB_IDP === "1") return issueAuthCode(res, {
    client_id, redirect_uri, code_challenge, state, scope: scope ?? client.scope,
  }, "tester@example.com");

  const loginState = rnd(16);
  loginStates.set(loginState, {
    client_id, redirect_uri, code_challenge, state, scope: scope ?? client.scope,
    exp: now() + LOGIN_STATE_TTL,
  });
  const g = googleClient();
  const url = g.generateAuthUrl({
    access_type: "offline",
    scope: ["openid", "email"],
    state: loginState,
    prompt: "consent",
  });
  res.redirect(url);
});

function issueAuthCode(res, pending, email) {
  const authCode = rnd(16);
  authCodes.set(authCode, {
    client_id: pending.client_id,
    redirect_uri: pending.redirect_uri,
    code_challenge: pending.code_challenge,
    email,
    scope: pending.scope,
    exp: now() + CODE_TTL,
  });
  const back = new URL(pending.redirect_uri);
  back.searchParams.set("code", authCode);
  back.searchParams.set("state", pending.state ?? "");
  res.redirect(back.toString());
}

app.get("/callback", async (req, res) => {
  const { code, state, error } = req.query;
  if (error) return res.status(400).send(`Google error: ${error}`);
  const pending = pick(loginStates, state);
  if (!pending || pending.exp < now()) return res.status(400).send("expired login state");

  const g = googleClient();
  const { tokens } = await g.getToken(code);
  const ticket = await g.verifyIdToken({ idToken: tokens.id_token, audience: GOOGLE_CLIENT_ID });
  const email = ticket.getPayload().email;
  if (!email) return res.status(400).send("Google did not return an email");
  issueAuthCode(res, pending, email);
});

// --- token endpoint -------------------------------------------------------

app.post("/token", async (req, res) => {
  const { grant_type } = req.body;
  try {
    if (grant_type === "authorization_code") {
      const { code, redirect_uri, code_verifier, client_id } = req.body;
      const entry = pick(authCodes, code);
      if (!entry || entry.exp < now()) throw oauthError("invalid_grant", "bad code");
      if (entry.client_id !== client_id) throw oauthError("invalid_grant", "client mismatch");
      if (redirect_uri && entry.redirect_uri && redirect_uri !== entry.redirect_uri) {
        throw oauthError("invalid_grant", "redirect mismatch");
      }
      if (!code_verifier) throw oauthError("invalid_grant", "code_verifier required");
      const expected = b64url(sha256(code_verifier));
      if (expected !== entry.code_challenge) throw oauthError("invalid_grant", "PKCE failed");
      const tokens = await issueTokens({ email: entry.email, scope: entry.scope });
      return res.json(tokens);
    }
    if (grant_type === "refresh_token") {
      const { refresh_token, scope } = req.body;
      const tokens = await rotateRefresh(refresh_token, scope);
      return res.json(tokens);
    }
    throw oauthError("unsupported_grant_type", grant_type);
  } catch (e) {
    const { code, desc } = e.oauth ?? { code: "invalid_grant", desc: e.message };
    res.status(400).json({ error: code, error_description: desc });
  }
});

app.post("/revoke", (req, res) => {
  // ponytail: best-effort — verify + revoke grant if it's a refresh token.
  verifyJwt(req.body.token).then((p) => { if (p.typ === "refresh") revokeGrant(p.grant); }).catch(() => {});
  res.status(200).end();
});

// --- protected MCP endpoint (R6: verified on every call) ------------------

async function authenticate(req) {
  const auth = req.headers.authorization;
  if (!auth?.startsWith("Bearer ")) return null;
  try {
    const payload = await verifyJwt(auth.slice(7));
    if (payload.typ !== "access") return null;
    return payload;
  } catch {
    return null;
  }
}

function unauthorized(res) {
  res.set("WWW-Authenticate", `Bearer resource_metadata="${ISSUER}/.well-known/oauth-protected-resource"`);
  res.status(401).json({ error: "invalid_token" });
}

async function handleMcp(req, res) {
  const user = await authenticate(req);
  if (!user) return unauthorized(res);

  const server = new McpServer({ name: "mcp-auth-glm", version: "1.0.0" }, { capabilities: { logging: {} } });
  // The demo tool: returns the real logged-in identity (R3, R6, acceptance #4).
  server.registerTool(
    "whoami",
    { description: "Returns the authenticated user's email, proving the token was verified." },
    async () => ({
      content: [{ type: "text", text: JSON.stringify({ email: user.email, sub: user.sub, scope: user.scope }) }],
    }),
  );
  server.registerTool(
    "echo",
    { description: "Echoes a message back, proving the token is reused without re-login (R2, R5).",
      inputSchema: { message: z.string() } },
    async ({ message }) => ({ content: [{ type: "text", text: `${user.email}: ${message}` }] }),
  );

  const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined });
  await server.connect(transport);
  req.auth = { token: user, extra: { sub: user.email } };
  await transport.handleRequest(req, res, req.body);
  res.on("close", () => { transport.close(); server.close(); });
}

app.post("/mcp", handleMcp);
app.get("/mcp", async (_req, res) => {
  // Streamable HTTP reserves GET for server→client streaming; no sessions here.
  res.writeHead(405).end(JSON.stringify({ jsonrpc: "2.0", error: { code: -32000, message: "Method not allowed." }, id: null }));
});
app.delete("/mcp", (_req, res) => res.status(405).end());

app.get("/health", (_req, res) => res.json({ ok: true, issuer: ISSUER }));

app.listen(PORT, () => {
  console.log(`mcp-auth-glm on :${PORT}  issuer=${ISSUER}`);
  console.log(`google redirect_uri=${GOOGLE_REDIRECT_URI}`);
  if (!GOOGLE_CLIENT_ID) console.warn("WARNING: GOOGLE_CLIENT_ID not set — /authorize will fail.");
});
