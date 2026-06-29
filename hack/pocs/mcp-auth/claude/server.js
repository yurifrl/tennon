// MCP server with Notion-style browser login, federated to Google.
//
// We are our OWN OAuth 2.1 authorization server (Google has no DCR / RFC 7591),
// so MCP clients DCR + PKCE against us, and we federate the human login to Google.
// The MCP SDK's mcpAuthRouter gives us RFC 9728 / 8414 / 7591 / 6749 / 7636 for free;
// we only implement the OAuthServerProvider hooks and a Google callback route.

import crypto from 'node:crypto';
import fs from 'node:fs';
import express from 'express';
import { SignJWT, jwtVerify, decodeJwt } from 'jose';
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import { mcpAuthRouter, getOAuthProtectedResourceMetadataUrl } from '@modelcontextprotocol/sdk/server/auth/router.js';
import { requireBearerAuth } from '@modelcontextprotocol/sdk/server/auth/middleware/bearerAuth.js';
import { z } from 'zod';

// Load tennon/.env (Google creds) if not already in the environment, so the server
// works no matter how it's launched (task, bare `node server.js`, etc.).
if (!process.env.GOOGLE_CLIENT_ID) {
  try { process.loadEnvFile(new URL('../../../../.env', import.meta.url)); } catch {}
}

const PORT = Number(process.env.PORT || 8000);
const PUBLIC_URL = (process.env.PUBLIC_URL || `http://localhost:${PORT}`).replace(/\/$/, '');
// Real IdP when Google creds are present; otherwise a built-in fake login page (dev mode).
const DEV_MODE = !process.env.GOOGLE_CLIENT_ID;
const GOOGLE_CLIENT_ID = DEV_MODE ? '' : process.env.GOOGLE_CLIENT_ID;
const GOOGLE_CLIENT_SECRET = DEV_MODE ? '' : required('GOOGLE_CLIENT_SECRET');
const JWT_SECRET = new TextEncoder().encode(process.env.JWT_SECRET || crypto.randomBytes(32).toString('hex'));

const ISSUER = new URL(PUBLIC_URL);
const RESOURCE = new URL('/mcp', PUBLIC_URL);
const GOOGLE_REDIRECT = `${PUBLIC_URL}/oauth/callback`;
const ACCESS_TTL = '1h';
const REFRESH_TTL = '30d'; // ponytail: absolute, no sliding window for a POC

function required(name) {
  const v = process.env[name];
  if (!v) throw new Error(`Missing required env ${name}`);
  return v;
}

// ponytail: persist registered clients to disk so restarts don't invalidate cached client_ids.
const CLIENTS_FILE = new URL('./clients.json', import.meta.url);
const clients = new Map();      // client_id -> OAuthClientInformationFull
try { for (const [k, v] of Object.entries(JSON.parse(fs.readFileSync(CLIENTS_FILE, 'utf8')))) clients.set(k, v); } catch {}
const saveClients = () => fs.writeFileSync(CLIENTS_FILE, JSON.stringify(Object.fromEntries(clients), null, 2));
const logins = new Map();       // loginId -> { clientRedirectUri, clientState, codeChallenge, clientId }
const authCodes = new Map();    // code -> { email, codeChallenge, clientId, redirectUri }

async function signAccess(email, clientId, scopes) {
  return new SignJWT({ email, scope: scopes.join(' '), typ: 'access' })
    .setProtectedHeader({ alg: 'HS256' })
    .setSubject(email)
    .setAudience(clientId)
    .setIssuer(PUBLIC_URL)
    .setIssuedAt()
    .setExpirationTime(ACCESS_TTL)
    .sign(JWT_SECRET);
}

async function signRefresh(email, clientId, scopes) {
  return new SignJWT({ email, scope: scopes.join(' '), typ: 'refresh' })
    .setProtectedHeader({ alg: 'HS256' })
    .setSubject(email)
    .setAudience(clientId)
    .setIssuer(PUBLIC_URL)
    .setIssuedAt()
    .setExpirationTime(REFRESH_TTL)
    .sign(JWT_SECRET);
}

async function issueTokens(email, clientId, scopes) {
  return {
    access_token: await signAccess(email, clientId, scopes),
    token_type: 'bearer',
    expires_in: 3600,
    refresh_token: await signRefresh(email, clientId, scopes),
    scope: scopes.join(' ')
  };
}

const provider = {
  clientsStore: {
    getClient: (id) => clients.get(id),
    registerClient: (client) => {
      clients.set(client.client_id, client);
      saveClients();
      return client;
    }
  },

  // Begin the flow: stash the client's PKCE challenge, send the human to the IdP.
  async authorize(client, params, res) {
    const loginId = crypto.randomUUID();
    logins.set(loginId, {
      clientRedirectUri: params.redirectUri,
      clientState: params.state,
      codeChallenge: params.codeChallenge,
      clientId: client.client_id,
      scopes: params.scopes ?? []
    });
    if (DEV_MODE) return res.redirect(302, `/dev/login?login=${loginId}`);
    const url = new URL('https://accounts.google.com/o/oauth2/v2/auth');
    url.searchParams.set('client_id', GOOGLE_CLIENT_ID);
    url.searchParams.set('redirect_uri', GOOGLE_REDIRECT);
    url.searchParams.set('response_type', 'code');
    url.searchParams.set('scope', 'openid email');
    url.searchParams.set('state', loginId);
    res.redirect(302, url.href);
  },

  async challengeForAuthorizationCode(_client, code) {
    const entry = authCodes.get(code);
    if (!entry) throw new Error('invalid authorization code');
    return entry.codeChallenge;
  },

  async exchangeAuthorizationCode(client, code) {
    const entry = authCodes.get(code);
    if (!entry || entry.clientId !== client.client_id) throw new Error('invalid authorization code');
    authCodes.delete(code); // one-time use
    return issueTokens(entry.email, client.client_id, entry.scopes ?? []);
  },

  // ponytail: rotate-on-refresh without reuse detection. Add a revocation set if abuse matters.
  async exchangeRefreshToken(client, refreshToken, scopes) {
    const { payload } = await jwtVerify(refreshToken, JWT_SECRET, { issuer: PUBLIC_URL, audience: client.client_id });
    if (payload.typ !== 'refresh') throw new Error('not a refresh token');
    const granted = (payload.scope || '').split(' ').filter(Boolean);
    return issueTokens(payload.email, client.client_id, scopes?.length ? scopes : granted);
  },

  async verifyAccessToken(token) {
    const { payload } = await jwtVerify(token, JWT_SECRET, { issuer: PUBLIC_URL });
    if (payload.typ !== 'access') throw new Error('not an access token');
    return {
      token,
      clientId: payload.aud,
      scopes: (payload.scope || '').split(' ').filter(Boolean),
      expiresAt: payload.exp,
      extra: { email: payload.email }
    };
  }
};

const app = express();

// RFC 9728 + 8414 metadata, RFC 7591 DCR, RFC 6749/7636 authorize+token endpoints.
app.use(mcpAuthRouter({
  provider,
  issuerUrl: ISSUER,
  resourceServerUrl: RESOURCE,
  scopesSupported: ['openid', 'email'],
  resourceName: 'MCP Auth POC'
}));

// Mint our one-time auth code for a finished login and bounce back to the MCP client.
function completeLogin(login, email, res) {
  const code = crypto.randomBytes(24).toString('base64url');
  authCodes.set(code, { email, codeChallenge: login.codeChallenge, clientId: login.clientId, redirectUri: login.clientRedirectUri, scopes: login.scopes });
  const back = new URL(login.clientRedirectUri);
  back.searchParams.set('code', code);
  if (login.clientState) back.searchParams.set('state', login.clientState);
  res.redirect(302, back.href);
}

// Dev IdP: a one-field login page that stands in for Google. ponytail: no password, no real auth.
app.get('/dev/login', (req, res) => {
  const login = String(req.query.login || '');
  res.type('html').send(`<!doctype html><meta charset=utf-8><title>Sign in</title>
<form method=post action=/dev/login style="font-family:system-ui;max-width:20rem;margin:4rem auto">
  <h2>Dev sign-in</h2><p>No Google configured — enter an email to continue.</p>
  <input type=hidden name=login value="${login}">
  <input name=email type=email placeholder=you@example.com required style="width:100%;padding:.5rem">
  <button style="margin-top:1rem;padding:.5rem 1rem">Continue</button>
</form>`);
});

app.post('/dev/login', express.urlencoded({ extended: false }), (req, res) => {
  const login = logins.get(String(req.body.login));
  if (!login) return res.status(400).send('unknown login state');
  logins.delete(String(req.body.login));
  completeLogin(login, String(req.body.email), res);
});

// Google federation callback: exchange code -> id_token -> email, then mint our auth code.
app.get('/oauth/callback', async (req, res) => {
  try {
    const login = logins.get(String(req.query.state));
    if (!login) return res.status(400).send('unknown login state');
    logins.delete(String(req.query.state));

    const tokenRes = await fetch('https://oauth2.googleapis.com/token', {
      method: 'POST',
      headers: { 'content-type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        code: String(req.query.code),
        client_id: GOOGLE_CLIENT_ID,
        client_secret: GOOGLE_CLIENT_SECRET,
        redirect_uri: GOOGLE_REDIRECT,
        grant_type: 'authorization_code'
      })
    });
    if (!tokenRes.ok) return res.status(502).send(`google token exchange failed: ${await tokenRes.text()}`);
    const { id_token } = await tokenRes.json();
    const email = decodeJwt(id_token).email; // id_token came straight from Google over TLS
    completeLogin(login, email, res);
  } catch (err) {
    res.status(500).send(`callback error: ${err.message}`);
  }
});

// The MCP endpoint, guarded by bearer auth (RFC 9728 WWW-Authenticate on 401).
const requireAuth = requireBearerAuth({
  verifier: provider,
  resourceMetadataUrl: getOAuthProtectedResourceMetadataUrl(RESOURCE)
});

app.post('/mcp', requireAuth, async (req, res) => {
  const email = req.auth?.extra?.email;
  const server = new McpServer({ name: 'mcp-auth-poc', version: '1.0.0' });
  server.tool('whoami', 'Return the authenticated user identity', {}, async () => ({
    content: [{ type: 'text', text: `You are authenticated as ${email}` }]
  }));
  const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined });
  res.on('close', () => transport.close());
  await server.connect(transport);
  await transport.handleRequest(req, res);
});

app.listen(PORT, () => {
  console.log(`MCP auth POC on ${PUBLIC_URL}${DEV_MODE ? '  [DEV IdP: no Google, fake login page]' : ''}`);
  console.log(`  resource metadata: ${getOAuthProtectedResourceMetadataUrl(RESOURCE)}`);
  console.log(`  add this URL to your MCP client: ${RESOURCE.href}`);
});
