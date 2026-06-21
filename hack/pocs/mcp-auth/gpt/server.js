import crypto from 'node:crypto';
import express from 'express';
import { SignJWT, jwtVerify, decodeJwt } from 'jose';
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import { mcpAuthRouter, getOAuthProtectedResourceMetadataUrl } from '@modelcontextprotocol/sdk/server/auth/router.js';
import { requireBearerAuth } from '@modelcontextprotocol/sdk/server/auth/middleware/bearerAuth.js';

const PORT = Number(process.env.PORT || 8000);
const PUBLIC_URL = (process.env.PUBLIC_URL || `http://localhost:${PORT}`).replace(/\/$/, '');
const DEV_MODE = !process.env.GOOGLE_CLIENT_ID;
const GOOGLE_CLIENT_ID = process.env.GOOGLE_CLIENT_ID || '';
const GOOGLE_CLIENT_SECRET = process.env.GOOGLE_CLIENT_SECRET || '';
const JWT_SECRET = new TextEncoder().encode(process.env.JWT_SECRET || crypto.randomBytes(32).toString('hex'));

const ISSUER = new URL(PUBLIC_URL);
const RESOURCE = new URL('/mcp', PUBLIC_URL);
const GOOGLE_REDIRECT = `${PUBLIC_URL}/google/callback`;
const clients = new Map();
const logins = new Map();
const authCodes = new Map();

async function signToken(email, clientId, scopes, typ, expires) {
  return new SignJWT({ email, scope: scopes.join(' '), typ })
    .setProtectedHeader({ alg: 'HS256' })
    .setSubject(email)
    .setAudience(clientId)
    .setIssuer(PUBLIC_URL)
    .setIssuedAt()
    .setExpirationTime(expires)
    .sign(JWT_SECRET);
}

async function issueTokens(email, clientId, scopes) {
  return {
    access_token: await signToken(email, clientId, scopes, 'access', '1h'),
    token_type: 'bearer',
    expires_in: 3600,
    refresh_token: await signToken(email, clientId, scopes, 'refresh', '30d'),
    scope: scopes.join(' ')
  };
}

const provider = {
  clientsStore: {
    getClient: id => clients.get(id),
    registerClient: client => {
      clients.set(client.client_id, client);
      return client;
    }
  },

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
    return res.redirect(302, url.href);
  },

  async challengeForAuthorizationCode(_client, code) {
    const entry = authCodes.get(code);
    if (!entry) throw new Error('invalid authorization code');
    return entry.codeChallenge;
  },

  async exchangeAuthorizationCode(client, code) {
    const entry = authCodes.get(code);
    if (!entry || entry.clientId !== client.client_id) throw new Error('invalid authorization code');
    authCodes.delete(code);
    return issueTokens(entry.email, client.client_id, entry.scopes ?? []);
  },

  async exchangeRefreshToken(client, refreshToken, scopes) {
    const { payload } = await jwtVerify(refreshToken, JWT_SECRET, { issuer: PUBLIC_URL, audience: client.client_id });
    if (payload.typ !== 'refresh') throw new Error('not a refresh token');
    const granted = String(payload.scope || '').split(' ').filter(Boolean);
    return issueTokens(payload.email, client.client_id, scopes?.length ? scopes : granted);
  },

  async verifyAccessToken(token) {
    const { payload } = await jwtVerify(token, JWT_SECRET, { issuer: PUBLIC_URL });
    if (payload.typ !== 'access') throw new Error('not an access token');
    return {
      token,
      clientId: payload.aud,
      scopes: String(payload.scope || '').split(' ').filter(Boolean),
      expiresAt: payload.exp,
      extra: { email: payload.email }
    };
  }
};

const app = express();

app.get('/.well-known/oauth-protected-resource', (_req, res) => {
  res.json({
    resource: RESOURCE.href.replace(/\/$/, ''),
    authorization_servers: [PUBLIC_URL],
    bearer_methods_supported: ['header'],
    scopes_supported: ['openid', 'email', 'mcp']
  });
});

app.use(mcpAuthRouter({
  provider,
  issuerUrl: ISSUER,
  resourceServerUrl: RESOURCE,
  scopesSupported: ['openid', 'email', 'mcp'],
  resourceName: 'MCP Auth GPT POC'
}));

function completeLogin(login, email, res) {
  const code = crypto.randomBytes(24).toString('base64url');
  authCodes.set(code, { email, codeChallenge: login.codeChallenge, clientId: login.clientId, scopes: login.scopes });
  const back = new URL(login.clientRedirectUri);
  back.searchParams.set('code', code);
  if (login.clientState) back.searchParams.set('state', login.clientState);
  res.redirect(302, back.href);
}

app.get('/dev/login', (req, res) => {
  res.type('html').send(`<!doctype html><title>Sign in</title><form method="post" action="/dev/login"><input type="hidden" name="login" value="${String(req.query.login || '')}"><input name="email" type="email" required><button>Continue</button></form>`);
});

app.post('/dev/login', express.urlencoded({ extended: false }), (req, res) => {
  const login = logins.get(String(req.body.login));
  if (!login) return res.status(400).send('unknown login state');
  logins.delete(String(req.body.login));
  completeLogin(login, String(req.body.email), res);
});

app.get('/google/callback', async (req, res) => {
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
    if (!tokenRes.ok) return res.status(502).send(await tokenRes.text());
    completeLogin(login, decodeJwt((await tokenRes.json()).id_token).email, res);
  } catch (err) {
    res.status(500).send(err.message);
  }
});

const requireAuth = requireBearerAuth({
  verifier: provider,
  resourceMetadataUrl: getOAuthProtectedResourceMetadataUrl(RESOURCE)
});

app.post('/mcp', requireAuth, async (req, res) => {
  const server = new McpServer({ name: 'mcp-auth-gpt', version: '1.0.0' });
  server.tool('whoami', 'Return the authenticated user identity', {}, async () => ({
    content: [{ type: 'text', text: `You are authenticated as ${req.auth.extra.email}` }]
  }));
  const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined });
  res.on('close', () => transport.close());
  await server.connect(transport);
  await transport.handleRequest(req, res);
});

app.listen(PORT, () => {
  console.log(`MCP auth GPT POC on ${PUBLIC_URL}`);
  console.log(`Add this URL to your MCP client: ${RESOURCE.href}`);
});
