#!/usr/bin/env node
import http from 'http';
import url from 'url';
import crypto from 'crypto';

const PORT = process.env.PORT || 8787;
const BASE_URL = process.env.BASE_URL || `http://localhost:${PORT}`;

// Google OAuth redirect URI - can be configured separately from BASE_URL
// This allows using a pre-registered redirect URI while the server runs on a different port
const GOOGLE_REDIRECT_URI = process.env.GOOGLE_REDIRECT_URI || `${BASE_URL}/oauth/callback`;

// Google OAuth config
const GOOGLE_CLIENT_ID = process.env.GOOGLE_CLIENT_ID;
const GOOGLE_CLIENT_SECRET = process.env.GOOGLE_CLIENT_SECRET;
const USE_REAL_GOOGLE = !!(GOOGLE_CLIENT_ID && GOOGLE_CLIENT_SECRET);

// In-memory stores (ponytail: POC only)
const clients = new Map();
const tokens = new Map();
const pendingCodes = new Map();

function generateToken() {
  return crypto.randomUUID();
}

// ===== OAUTH =====

function getProtectedResourceMetadata() {
  return {
    resource: BASE_URL,
    authorization_servers: [BASE_URL],
    scopes_supported: ['openid', 'email', 'profile'],
    bearer_methods_supported: ['header']
  };
}

function getAuthorizationServerMetadata() {
  return {
    issuer: BASE_URL,
    authorization_endpoint: `${BASE_URL}/oauth/authorize`,
    token_endpoint: `${BASE_URL}/oauth/token`,
    registration_endpoint: `${BASE_URL}/oauth/register`,
    scopes_supported: ['openid', 'email', 'profile'],
    response_types_supported: ['code'],
    code_challenge_methods_supported: ['S256'],
    token_endpoint_auth_methods_supported: ['none'],
    grant_types_supported: ['authorization_code']
  };
}

function registerClient(metadata) {
  const clientId = crypto.randomUUID();
  const client = {
    client_id: clientId,
    client_secret: crypto.randomBytes(32).toString('base64url'),
    token_endpoint_auth_method: 'none',
    redirect_uris: metadata.redirect_uris || ['http://localhost/callback'],
    grant_types: ['authorization_code'],
    response_types: ['code']
  };
  clients.set(clientId, client);
  return client;
}

async function exchangeGoogleCode(code) {
  const tokenRes = await fetch('https://oauth2.googleapis.com/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      code,
      client_id: GOOGLE_CLIENT_ID,
      client_secret: GOOGLE_CLIENT_SECRET,
      redirect_uri: GOOGLE_REDIRECT_URI,
      grant_type: 'authorization_code'
    })
  });
  const tokens = await tokenRes.json();
  if (tokens.error) throw new Error(tokens.error);

  const userRes = await fetch('https://openidconnect.googleapis.com/v1/userinfo', {
    headers: { Authorization: `Bearer ${tokens.access_token}` }
  });
  const userInfo = await userRes.json();
  return userInfo;
}

// ===== MCP =====

async function handleMcpRequest(body, token) {
  const user = tokens.get(token);
  if (!user) {
    return { error: 'invalid_token', message: 'Token expired or invalid' };
  }

  const msg = JSON.parse(body || '{}');
  const id = msg.id;

  if (msg.method === 'initialize') {
    return {
      jsonrpc: '2.0',
      id,
      result: {
        protocolVersion: '2024-11-05',
        capabilities: { tools: {} },
        serverInfo: { name: 'mcp-auth-poc', version: '1.0.0' }
      }
    };
  }

  if (msg.method === 'tools/list') {
    return {
      jsonrpc: '2.0',
      id,
      result: {
        tools: [{
          name: 'whoami',
          description: 'Returns authenticated user email',
          inputSchema: { type: 'object' }
        }]
      }
    };
  }

  if (msg.method === 'tools/call') {
    if (msg.params?.name === 'whoami') {
      return {
        jsonrpc: '2.0',
        id,
        result: {
          content: [{ type: 'text', text: `Authenticated as: ${user.email}` }],
          isError: false
        }
      };
    }
  }

  return { jsonrpc: '2.0', id, error: { code: -32601, message: 'Method not found' } };
}

// ===== MAIN SERVER =====

async function readBody(req) {
  return new Promise((resolve) => {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', () => resolve(body));
  });
}

const server = http.createServer(async (req, res) => {
  const parsed = url.parse(req.url, true);
  const pathname = parsed.pathname;

  // CORS
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization, MCP-SESSION-ID');

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  try {
    // RFC 9728: Protected Resource Metadata
    if (pathname === '/.well-known/oauth-protected-resource') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(getProtectedResourceMetadata()));
      return;
    }

    // RFC 8414: Authorization Server Metadata
    if (pathname === '/.well-known/oauth-authorization-server') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(getAuthorizationServerMetadata()));
      return;
    }

    // RFC 7591: Dynamic Client Registration
    if (pathname === '/oauth/register' && req.method === 'POST') {
      const body = JSON.parse(await readBody(req));
      const client = registerClient(body);
      res.writeHead(201, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(client));
      return;
    }

    // OAuth Authorization Endpoint
    if (pathname === '/oauth/authorize') {
      const { client_id, redirect_uri, code_challenge, state } = parsed.query;
      const internalCode = crypto.randomUUID();

      pendingCodes.set(internalCode, {
        clientId: client_id,
        redirectUri: redirect_uri,
        codeChallenge: code_challenge,
        state,
        userInfo: { email: 'demo@example.com', name: 'Demo User', sub: 'demo123' }
      });

      if (USE_REAL_GOOGLE) {
        const params = new URLSearchParams({
          client_id: GOOGLE_CLIENT_ID,
          response_type: 'code',
          scope: 'openid email profile',
          redirect_uri: GOOGLE_REDIRECT_URI,
          state: internalCode,
          access_type: 'offline', prompt: 'consent'
        });
        res.writeHead(302, { Location: `https://accounts.google.com/o/oauth2/v2/auth?${params}` });
        res.end();
      } else {
        // Demo mode: auto-approve
        const target = new URL(redirect_uri);
        target.searchParams.set('code', internalCode);
        if (state) target.searchParams.set('state', state);
        res.writeHead(302, { Location: target.toString() });
        res.end();
      }
      return;
    }

    // Google OAuth Callback
    if (pathname === '/oauth/callback') {
      const { code, state: internalCode, error } = parsed.query;
      if (error) {
        res.writeHead(400); res.end(`Google error: ${error}`); return;
      }

      const pending = pendingCodes.get(internalCode);
      if (!pending) {
        res.writeHead(400); res.end('Invalid or expired code'); return;
      }

      try {
        let userInfo;
        if (USE_REAL_GOOGLE) {
          userInfo = await exchangeGoogleCode(code);
        } else {
          userInfo = { email: 'demo@example.com', name: 'Demo User', sub: 'demo123' };
        }

        // Create our token
        const token = generateToken();
        tokens.set(token, userInfo);

        // Return via redirect
        const target = new URL(pending.redirectUri);
        // Exchange authorization code for token
        const authCode = crypto.randomUUID();
        pendingCodes.set(authCode, { userInfo, redirectUri: pending.redirectUri, state: pending.state });
        target.searchParams.set('code', authCode);
        if (pending.state) target.searchParams.set('state', pending.state);
        res.writeHead(302, { Location: target.toString() });

      } catch (e) {
        res.writeHead(400); res.end(`OAuth error: ${e.message}`);
      }
      return;
    }

    // OAuth Token Endpoint
    if (pathname === '/oauth/token' && req.method === 'POST') {
      const body = await readBody(req);
      const params = new URLSearchParams(body);
      const code = params.get('code');
      const codeVerifier = params.get('code_verifier');

      const pending = pendingCodes.get(code);
      if (!pending) {
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'invalid_grant' }));
        return;
      }

      // Create token
      const token = generateToken();
      tokens.set(token, pending.userInfo);
      pendingCodes.delete(code);

      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        access_token: token,
        token_type: 'Bearer',
        expires_in: 3600,
        scope: 'openid email profile'
      }));
      return;
    }

    // MCP Endpoint - Protected (Streamable HTTP)
    if (pathname === '/mcp') {
      const authHeader = req.headers.authorization;
      const token = authHeader?.replace(/^Bearer\s+/i, '');

      // RFC 9728: Return 401 with resource metadata BEFORE checking Accept (client needs to auth first)
      if (!token) {
        res.writeHead(401, {
          'Content-Type': 'application/json',
          'WWW-Authenticate': `Bearer resource_metadata="${BASE_URL}"`
        });
        res.end(JSON.stringify({
          jsonrpc: '2.0',
          id: null,
          error: { code: -32000, message: 'Authentication required' }
        }));
        return;
      }

      const user = tokens.get(token);
      if (!user) {
        res.writeHead(401, {
          'Content-Type': 'application/json',
          'WWW-Authenticate': `Bearer resource_metadata="${BASE_URL}"`
        });
        res.end(JSON.stringify({
          jsonrpc: '2.0',
          id: null,
          error: { code: -32000, message: 'Token expired or invalid' }
        }));
        return;
      }

      // Check Accept header for streamable HTTP (only after auth)
      const accept = req.headers.accept || '';
      if (!accept.includes('application/json') || !accept.includes('text/event-stream')) {
        res.writeHead(406, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
          jsonrpc: '2.0',
          id: null,
          error: { code: -32000, message: 'Not Acceptable: Client must accept both application/json and text/event-stream' }
        }));
        return;
      }

      const body = await readBody(req);
      const result = await handleMcpRequest(body, token);

      res.writeHead(200, {
        'Content-Type': 'application/json',
        'Mcp-Session-Id': crypto.randomUUID()
      });
      res.end(JSON.stringify(result));
      return;
    }

    res.writeHead(404); res.end('Not found');

  } catch (err) {
    console.error('Error:', err);
    res.writeHead(500);
    res.end('Internal error');
  }
});

server.listen(PORT, () => {
  console.log(`MCP Auth POC: ${BASE_URL}`);
  console.log('Endpoints:');
  console.log(`  GET /.well-known/oauth-protected-resource`);
  console.log(`  GET /.well-known/oauth-authorization-server`);
  console.log(`  POST /oauth/register`);
  console.log(`  GET /oauth/authorize`);
  console.log(`  POST /oauth/token`);
  console.log(`  POST /mcp (protected)`);
  if (!USE_REAL_GOOGLE) {
    console.log('\nDEMO MODE: Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET for real auth');
  }
});
